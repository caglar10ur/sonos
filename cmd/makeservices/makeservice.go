// Command makeservice generates a Go client for a UPnP service from its
// SCPD (service control protocol description) XML.
package main

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"unicode"
)

type AllowedValueRange struct {
	XMLName xml.Name `xml:"allowedValueRange"`
	Minimum string   `xml:"minimum"`
	Maximum string   `xml:"maximum"`
	Step    string   `xml:"step"`
}

type StateVariable struct {
	XMLName           xml.Name           `xml:"stateVariable"`
	SendEvents        string             `xml:"sendEvents,attr"`
	Multicast         string             `xml:"multicast,attr"`
	Name              string             `xml:"name"`
	DataType          string             `xml:"dataType"`
	DefaultValue      string             `xml:"defaultValue"`
	AllowedValueRange *AllowedValueRange `xml:"allowedValueRange"`
	AllowedValues     []string           `xml:"allowedValueList>allowedValue"`
}

// UnderlyingGoType is the Go type of the variable's UPnP data type, or ""
// if the data type is not supported.
func (s *StateVariable) UnderlyingGoType() string {
	switch s.DataType {
	case "ui1":
		return "uint8"
	case "ui2":
		return "uint16"
	case "ui4":
		return "uint32"
	case "i1":
		return "int8"
	case "i2":
		return "int16"
	case "i4":
		return "int32"
	case "int":
		return "int64"
	case "r4":
		return "float32"
	case "number", "r8", "float", "float64":
		return "float64"
	case "boolean":
		return "bool"
	// char is a single Unicode character, sent as the character itself, and
	// uri is not decodable into *url.URL by encoding/xml, so both stay text.
	case "string", "char", "uri", "uuid",
		"date", "dateTime", "dateTime.tz", "time", "time.tz":
		return "string"
	default:
		return ""
	}
}

// GoName is the identifier the variable's generated type is based on.
func (s *StateVariable) GoName() string {
	return goName(s.Name)
}

// GoDataType is the Go type used for the variable in action arguments.
func (s *StateVariable) GoDataType() string {
	if len(s.AllowedValues) > 0 {
		return s.GoName() + "Enum"
	}
	return s.UnderlyingGoType()
}

// EventType is the Go type an evented state variable is reported as.
func (s *StateVariable) EventType() string {
	if len(s.AllowedValues) > 0 {
		return s.GoDataType()
	}
	return s.GoName()
}

// RangeDoc describes the allowed value range, e.g. "0 to 100", or returns
// "" if the variable has none.
func (s *StateVariable) RangeDoc() string {
	r := s.AllowedValueRange
	if r == nil || (r.Minimum == "" && r.Maximum == "") {
		return ""
	}
	doc := r.Minimum + " to " + r.Maximum
	if r.Step != "" && r.Step != "1" {
		doc += " in steps of " + r.Step
	}
	return doc
}

type Argument struct {
	XMLName              xml.Name `xml:"argument"`
	Name                 string   `xml:"name"`
	Direction            string   `xml:"direction"`
	RelatedStateVariable string   `xml:"relatedStateVariable"`
}

type Action struct {
	XMLName   xml.Name   `xml:"action"`
	Name      string     `xml:"name"`
	Arguments []Argument `xml:"argumentList>argument"`
}

type SpecVersion struct {
	XMLName xml.Name `xml:"specVersion"`
	Major   int      `xml:"major"`
	Minor   int      `xml:"minor"`
}

type Scpd struct {
	XMLName        xml.Name        `xml:"scpd"`
	Xmlns          string          `xml:"type,attr"`
	SpecVersion    SpecVersion     `xml:"specVersion"`
	StateVariables []StateVariable `xml:"serviceStateTable>stateVariable"`
	Actions        []Action        `xml:"actionList>action"`
}

// StateVariable returns the state variable with the given name, or nil.
func (s *Scpd) StateVariable(name string) *StateVariable {
	i := slices.IndexFunc(s.StateVariables, func(sv StateVariable) bool { return sv.Name == name })
	if i < 0 {
		return nil
	}
	return &s.StateVariables[i]
}

// EventedVariables returns the state variables that are sent in events.
func (s *Scpd) EventedVariables() []StateVariable {
	var out []StateVariable
	for _, sv := range s.StateVariables {
		if sv.SendEvents == "yes" {
			out = append(out, sv)
		}
	}
	return out
}

// HasEvents reports whether the service sends any events.
func (s *Scpd) HasEvents() bool {
	return len(s.EventedVariables()) > 0
}

// Validate checks that the template can turn the description into valid
// Go: every data type is supported, action arguments refer to existing state
// variables, enumerations are of strings, and every generated identifier is
// valid and unique within the package.
func (s *Scpd) Validate() error {
	// identifier -> what it was generated for, starting with the fixed
	// package-level names and Service methods the template always emits.
	idents := make(map[string]string)
	for _, fixed := range []string{"Service", "Option", "WithClient", "WithPaths", "NewService",
		"ServiceURN", "ControlPath", "EventPath", "propertySet", "rawProperty", "property", "propertyName"} {
		idents[fixed] = "the template"
	}
	methods := map[string]string{"ControlEndpoint": "the template", "EventEndpoint": "the template", "ParseEvent": "the template"}
	declare := func(ident, what string) error {
		if !token.IsIdentifier(ident) {
			return fmt.Errorf("%s: %q is not a valid Go identifier", what, ident)
		}
		if prev, ok := idents[ident]; ok {
			return fmt.Errorf("%s and %s both generate %s", prev, what, ident)
		}
		idents[ident] = what
		return nil
	}

	for _, sv := range s.StateVariables {
		what := fmt.Sprintf("state variable %q", sv.Name)
		if sv.UnderlyingGoType() == "" {
			return fmt.Errorf("%s: unsupported UPnP dataType %q", what, sv.DataType)
		}
		if len(sv.AllowedValues) > 0 {
			if sv.UnderlyingGoType() != "string" {
				return fmt.Errorf("%s: allowed values on non-string dataType %q", what, sv.DataType)
			}
			if err := declare(sv.GoDataType(), what); err != nil {
				return err
			}
			for _, v := range sv.AllowedValues {
				if err := declare(enumConst(sv.GoName(), v), fmt.Sprintf("%s value %q", what, v)); err != nil {
					return err
				}
			}
		} else if sv.SendEvents == "yes" {
			if err := declare(sv.EventType(), what); err != nil {
				return err
			}
		}
	}
	for _, a := range s.Actions {
		what := fmt.Sprintf("action %q", a.Name)
		if !token.IsIdentifier(a.Name) {
			return fmt.Errorf("%s: not a valid Go identifier", what)
		}
		if prev, ok := methods[a.Name]; ok {
			return fmt.Errorf("%s and %s both generate method %s", prev, what, a.Name)
		}
		methods[a.Name] = what
		for _, ident := range []string{a.Name + "Args", a.Name + "Response"} {
			if err := declare(ident, what); err != nil {
				return err
			}
		}
		for _, arg := range a.Arguments {
			if s.StateVariable(arg.RelatedStateVariable) == nil {
				return fmt.Errorf("%s argument %q: unknown related state variable %q", what, arg.Name, arg.RelatedStateVariable)
			}
			if !token.IsIdentifier(arg.Name) {
				return fmt.Errorf("%s argument %q: not a valid Go identifier", what, arg.Name)
			}
		}
	}
	return nil
}

// goName turns a state variable name into a Go identifier.
func goName(s string) string {
	s = strings.TrimPrefix(s, "A_ARG_TYPE_")
	return strings.Map(func(r rune) rune {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, s)
}

// enumConst names the constant for an allowed value, e.g.
// ("TransportState", "PAUSED_PLAYBACK") -> "TransportStatePausedPlayback".
// All-caps words are title-cased unless they are two letters or fewer, so
// "TRACK_NR" becomes "TrackNR" and "LF" stays "LF".
func enumConst(typeName, value string) string {
	var b strings.Builder
	b.WriteString(typeName)
	words := strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, w := range words {
		if len(w) > 2 && strings.ToUpper(w) == w {
			w = strings.ToLower(w)
		}
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String()
}

type tmplCtx struct {
	ServiceDefinition *Scpd
	ServiceName       string
	ControlEndpoint   string
	EventEndpoint     string
}

//go:embed service.tmpl
var serviceTemplate string

var tmpl = template.Must(template.New("service").Funcs(template.FuncMap{
	"lower":     strings.ToLower,
	"enumConst": enumConst,
}).Parse(serviceTemplate))

// dataTypeOverrides corrects state variables whose SCPD data type does not
// match what devices actually send, keyed by service and variable name.
var dataTypeOverrides = map[string]map[string]string{
	// Declared ui4, but sent as "RINCON_<uuid>,<n>" like FavoritesUpdateID.
	"ContentDirectory": {
		"RadioFavoritesUpdateID": "string",
		"RadioLocationUpdateID":  "string",
	},
}

// MakeServiceApi generates the Go source of a client for the service
// described by scpd.
func MakeServiceApi(serviceName, controlEndpoint, eventEndpoint string, scpd []byte) ([]byte, error) {
	var s Scpd
	if err := xml.Unmarshal(scpd, &s); err != nil {
		return nil, err
	}
	for i, v := range s.StateVariables {
		if t, ok := dataTypeOverrides[serviceName][v.Name]; ok {
			s.StateVariables[i].DataType = t
		}
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, tmplCtx{
		ServiceDefinition: &s,
		ServiceName:       serviceName,
		ControlEndpoint:   controlEndpoint,
		EventEndpoint:     eventEndpoint,
	})
	if err != nil {
		return nil, err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting generated code: %w", err)
	}
	return src, nil
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("makeservice: ")

	xmlPath := flag.String("xml", "", "path to the SCPD XML file, e.g. xml/AVTransport1.xml")
	controlEndpoint := flag.String("control", "", "control path of the service, e.g. /MediaRenderer/AVTransport/Control")
	eventEndpoint := flag.String("event", "", "event path of the service, e.g. /MediaRenderer/AVTransport/Event")
	outputDir := flag.String("outputDir", "services", "directory the <service>/<service>.go file is written under")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: makeservice -xml <file> -control <path> -event <path> [-outputDir <dir>]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *xmlPath == "" || *controlEndpoint == "" || *eventEndpoint == "" {
		flag.Usage()
		os.Exit(2)
	}

	scpd, err := os.ReadFile(*xmlPath)
	if err != nil {
		log.Fatal(err)
	}

	// The service name is the file name without the version suffix, e.g.
	// AlarmClock1.xml -> AlarmClock.
	serviceName := strings.TrimSuffix(filepath.Base(*xmlPath), filepath.Ext(*xmlPath))
	serviceName = strings.TrimSuffix(serviceName, "1")

	src, err := MakeServiceApi(serviceName, *controlEndpoint, *eventEndpoint, scpd)
	if err != nil {
		log.Fatalf("%s: %v", *xmlPath, err)
	}

	pkg := strings.ToLower(serviceName)
	dir := filepath.Join(*outputDir, pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatal(err)
	}
	out := filepath.Join(dir, pkg+".go")
	if err := os.WriteFile(out, src, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("generated", out)
}
