package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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

func (s *StateVariable) GoDataType() string {
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
	case "number", "r8":
		return "float"
	case "float", "float64":
		return "float"
	case "char":
		return "rune"
	case "string":
		return "string"
	case "date", "dateTime", " dateTime.tz", "time", "time.tz":
		return "string"
	case "boolean":
		return "bool"
	case "uri":
		return "*url.URL"
	case "uuid":
		return "string"
	default:
		return ""
	}
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

func (s *Scpd) GetStateVariable(name string) *StateVariable {
	for _, sv := range s.StateVariables {
		if sv.Name == name {
			return &sv
		}
	}
	return nil
}

type tmplCtx struct {
	ServiceDefinition *Scpd
	ServiceName       string
	ServiceEndpoint   string
	ControlEndpoint   string
}

func MakeServiceApi(serviceName, serviceControlEndpoint, serviceEventEndpoint string, scdp []byte) ([]byte, error) {
	var s Scpd
	err := xml.Unmarshal(scdp, &s)
	if err != nil {
		return nil, err
	}

	ctx := tmplCtx{
		ServiceDefinition: &s,
		ServiceName:       serviceName,
		ServiceEndpoint:   serviceControlEndpoint,
		ControlEndpoint:   serviceEventEndpoint,
	}

	tpl, err := template.New("service").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).Parse(serviceTpl)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBufferString("")
	err = tpl.Execute(buf, ctx)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func main() {
	xmlPath := flag.String("xml", "", "Path to the XML service definition file")
	controlEndpoint := flag.String("control", "", "Service control endpoint URL")
	eventEndpoint := flag.String("event", "", "Service event endpoint URL")
	outputDir := flag.String("outputDir", "services", "Output directory for the generated Go file")

	flag.Parse()

	if *xmlPath == "" || *controlEndpoint == "" || *eventEndpoint == "" {
		fmt.Println("Usage: makeservice -xml <path_to_xml> -control <control_url> -event <event_url> [-outputDir <output_directory>]")
		os.Exit(1)
	}

	// Read the XML file
	scdp, err := os.ReadFile(*xmlPath)
	if err != nil {
		fmt.Printf("Error reading XML file: %v\n", err)
		os.Exit(1)
	}

	// Extract service name from XML filename
	baseName := filepath.Base(*xmlPath)
	serviceName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	serviceName = strings.TrimSuffix(serviceName, "1") // Remove trailing '1' if present (e.g., AlarmClock1 -> AlarmClock)

	// Generate the service API code
	generatedCode, err := MakeServiceApi(serviceName, *controlEndpoint, *eventEndpoint, scdp)
	if err != nil {
		fmt.Printf("Error generating service API: %v\n", err)
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	finalOutputDir := filepath.Join(*outputDir, serviceName)
	if err := os.MkdirAll(finalOutputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory %s: %v\n", finalOutputDir, err)
		os.Exit(1)
	}

	// Write the generated code to a file
	outputFileName := filepath.Join(finalOutputDir, serviceName+".go")
	if err := os.WriteFile(outputFileName, generatedCode, 0644); err != nil {
		fmt.Printf("Error writing generated code to file %s: %v\n", outputFileName, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated %s\n", outputFileName)
}

// MakeServiceApi generates the Go service API code based on the provided SCDP XML.
// (The rest of the MakeServiceApi function and other structs/constants remain unchanged)

const serviceTpl = `// Code generated by makeservice based on the provided SCDP XML. DO NOT EDIT.

// Package {{.ServiceName | lower }} is a generated {{.ServiceName}} package.
package {{.ServiceName | lower }}

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
)

const (
	ServiceURN     = "urn:schemas-upnp-org:service:{{.ServiceName}}:1"
	EncodingSchema = "http://schemas.xmlsoap.org/soap/encoding/"
	EnvelopeSchema = "http://schemas.xmlsoap.org/soap/envelope/"
)

type ServiceOption func(*Service)

func WithClient(c *http.Client) ServiceOption {
	return func(s *Service) {
		s.client = c
	}
}

func WithLocation(u *url.URL) ServiceOption {
	return func(s *Service) {
		s.location = u
	}
}

// State Variables
{{- range .ServiceDefinition.StateVariables}}
{{- if eq .SendEvents "yes"}}
type {{.Name}} {{.GoDataType}}
{{- end}}
{{- end}}

// Service represents {{.ServiceName}} service.
type Service struct {
	controlEndpoint *url.URL
	eventEndpoint   *url.URL
{{- range .ServiceDefinition.StateVariables}}
{{- if eq .SendEvents "yes"}}
	{{.Name}} *{{.Name}}
{{- end}}
{{- end}}
	location        *url.URL
	client          *http.Client
}

// NewService creates a new instance of the {{.ServiceName}} service.
// You must provide at least a location URL and an HTTP client using the options.
func NewService(opts ...ServiceOption) *Service {
	s := &Service{}

	c, err := url.Parse("{{.ServiceEndpoint}}")
	if nil != err {
		panic(err)
	}
	e, err := url.Parse("{{.ControlEndpoint}}")
	if nil != err {
		panic(err)
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.client == nil {
		panic("no client location")
	}
	if s.location == nil {
		panic("empty location")
	}

	s.controlEndpoint = s.location.ResolveReference(c)
	s.eventEndpoint = s.location.ResolveReference(e)

	return s
}

// ControlEndpoint returns the control endpoint URL of the service.
// This is usually a URL relative to the device's base URL.
func (s *Service) ControlEndpoint() *url.URL{
	return s.controlEndpoint
}

// EventEndpoint returns the event endpoint URL of the service.
// This is usually a URL relative to the device's base URL.
func (s *Service) EventEndpoint() *url.URL{
	return s.eventEndpoint
}

// Location returns the base URL of the device hosting the service.
func (s *Service) Location() *url.URL{
	return s.location
}

// Client returns the HTTP client used to communicate with the service.
func (s *Service) Client() *http.Client {
	return s.client
}

// internal use only
type envelope struct {
	XMLName xml.Name ` + "`" + `xml:"s:Envelope"` + "`" + `
	Xmlns string ` + "`" + `xml:"xmlns:s,attr"` + "`" + `
	EncodingStyle string ` + "`" + `xml:"s:encodingStyle,attr"` + "`" + `
	Body body ` + "`" + `xml:"s:Body"` + "`" + `
}

// internal use only
type body struct {
	XMLName xml.Name ` + "`" + `xml:"s:Body"` + "`" + `
{{- range .ServiceDefinition.Actions}}
	{{.Name}} *{{.Name}}Args ` + "`" + `xml:"u:{{.Name}},omitempty"` + "`" + `
{{- end}}
}

// internal use only
type envelopeResponse struct {
	XMLName xml.Name ` + "`" + `xml:"Envelope"` + "`" + `
	Xmlns string ` + "`" + `xml:"xmlns:s,attr"` + "`" + `
	EncodingStyle string ` + "`" + `xml:"encodingStyle,attr"` + "`" + `
	Body bodyResponse ` + "`" + `xml:"Body"` + "`" + `
}

// internal use only
type bodyResponse struct {
	XMLName xml.Name ` + "`" + `xml:"Body"` + "`" + `
{{- range .ServiceDefinition.Actions}}
	{{.Name}} *{{.Name}}Response ` + "`" + `xml:"{{.Name}}Response,omitempty"` + "`" + `
{{- end}}
}

func (s *Service) exec(actionName string, envelope *envelope) (*envelopeResponse, error) {
	postBody, err := xml.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", s.controlEndpoint.String(), bytes.NewBuffer(postBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=\"utf-8\"")
	req.Header.Set("SOAPAction", ServiceURN+"#"+actionName)
	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var envelopeResponse envelopeResponse
	err = xml.Unmarshal(responseBody, &envelopeResponse)
	if err != nil {
		return nil, err
	}
	return &envelopeResponse, nil
}

{{- range .ServiceDefinition.Actions}}
// {{.Name}} Argument type.
type {{.Name}}Args struct {
	Xmlns string ` + "`" + `xml:"xmlns:u,attr"` + "`" + `
{{- range .Arguments}}
{{- if eq .Direction "in"}}
	{{.Name}} {{ ($.ServiceDefinition.GetStateVariable .RelatedStateVariable).GoDataType }} ` + "`" + `xml:"{{.Name}}"` + "`" + `
{{- end}}
{{- end}}
}

// {{.Name}} Response type.
type {{.Name}}Response struct {
{{- range .Arguments}}
{{- if eq .Direction "out"}}
	{{.Name}} {{ ($.ServiceDefinition.GetStateVariable .RelatedStateVariable).GoDataType }} ` + "`" + `xml:"{{.Name}}"` + "`" + `
{{- end}}
{{- end}}
}

// {{.Name}} calls the {{.Name}} action on the service.
func (s *Service) {{.Name}}(args *{{.Name}}Args) (*{{.Name}}Response, error) {
	args.Xmlns = ServiceURN
	r, err := s.exec("{{.Name}}",
		&envelope{
			EncodingStyle: EncodingSchema,
			Xmlns:         EnvelopeSchema,
			Body:          body{ {{.Name}}: args },
		})
	if err != nil {
		return nil, err
	}
	if r.Body.{{.Name}} == nil {
		return nil, errors.New("unexpected response from service calling {{.Name}}()")
	}

	return r.Body.{{.Name}}, nil
}
{{end}}

// UpnpEvent represents a UPnP event notification.
type UpnpEvent struct {
	XMLName xml.Name ` + "`" + `xml:"propertyset"` + "`" + `
	XMLNameSpace string ` + "`" + `xml:"xmlns:e,attr"` + "`" + `
	Properties []Property ` + "`" + `xml:"property"` + "`" + `
}

// Property represents a single property in a UPnP event notification.
type Property struct {
	XMLName xml.Name ` + "`" + `xml:"property"` + "`" + `
{{- range .ServiceDefinition.StateVariables}}
{{- if eq .SendEvents "yes"}}
	{{.Name}} *{{.Name}} ` + "`" + `xml:"{{.Name}}"` + "`" + `
{{- end}}
{{- end}}
}

// ParseEvent parses a UPnP event notification and updates the service's state variables accordingly.
// It returns a slice of updated state variable values.
func (zp *Service) ParseEvent(body []byte) []interface{} {
	var evt UpnpEvent
	var events []interface{}

	if err := xml.Unmarshal(body, &evt); err != nil {
		return events
	}
	for _, prop := range evt.Properties {
		_ = prop
		switch {
		{{- range .ServiceDefinition.StateVariables}}
		{{- if eq .SendEvents "yes"}}
		case prop.{{.Name}} != nil:
			zp.{{.Name}} = prop.{{.Name}}
			events = append(events, *prop.{{.Name}})
		{{- end}}
		{{- end}}
		}
	}
	return events
}
`
