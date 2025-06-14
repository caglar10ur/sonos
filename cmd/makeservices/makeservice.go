package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

// TemplateData holds the data for generating the service code.
type TemplateData struct {
	ServiceName          string
	ServiceURN           string
	EncodingSchema       string
	EnvelopeSchema       string
	ControlEndpoint      string
	EventEndpoint        string
	StateVariables       []StateVariable
	EventStateVariables  []StateVariable // State variables with sendEvents="yes"
	Actions              []Action
	ServiceLowerName     string
	StateVariableStructs string
	ServiceStructFields  string
}

// GetStateVariable is a helper function for the template to find a state variable by name.
func (td *TemplateData) GetStateVariable(name string) *StateVariable {
	for i := range td.StateVariables {
		if td.StateVariables[i].Name == name {
			return &td.StateVariables[i]
		}
	}
	return nil
}

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
		return "float32" // Corrected typo and mapping
	case "number", "r8", "float", "float64":
		return "float64" // Consolidated mapping
	// case "fixed.14.4": // TODO fixed - Not yet supported
	case "char":
		return "rune"
	case "string":
		return "string"
		// TODO data/time - Not yet supported
	case "date", "dateTime", " dateTime.tz", "time", "time.tz":
		return "string" // Kept as string for now
	case "boolean":
		return "bool"
		// TODO - Not yet supported
	// case "bin.base64", "bin.hex":
	// 	return "string" // Kept as string for now
	case "uri":
		return "*url.URL"
	case "uuid":
		return "string" // Kept as string for now
	default:
		// Add comments for any unimplemented data types
		// For example:
		// case "customType":
		// return "" // Not yet supported
		return "" // Or handle as error
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

func MakeServiceApi(ServiceName, serviceControlEndpoint, serviceEventEndpoint string, scdp []byte) ([]byte, error) {
	var s Scpd
	err := xml.Unmarshal(scdp, &s)
	if err != nil {
		return nil, err
	}

	// Prepare data for the template.
	stateVariableStructs := bytes.NewBufferString("")
	serviceStructFields := bytes.NewBufferString("")
	eventStateVariables := []StateVariable{}
	for _, sv := range s.StateVariables {
		if sv.SendEvents == "yes" {
			fmt.Fprintf(stateVariableStructs, "type %s %s\n", sv.Name, sv.GoDataType())
			fmt.Fprintf(serviceStructFields, "%s *%s\n", sv.Name, sv.Name)
			eventStateVariables = append(eventStateVariables, sv)
		}
	}

	// Validate actions and their arguments.
	for _, action := range s.Actions {
		for _, argument := range action.Arguments {
			sv := s.GetStateVariable(argument.RelatedStateVariable)
			if sv == nil {
				return nil, fmt.Errorf("state variable %s not found for action %s argument %s", argument.RelatedStateVariable, action.Name, argument.Name)
			}
			// Ensure GoDataType is valid
			if sv.GoDataType() == "" {
				return nil, fmt.Errorf("unsupported data type %s for state variable %s", sv.DataType, sv.Name)
			}
		}
	}

	data := TemplateData{
		ServiceName:          ServiceName,
		ServiceURN:           "urn:schemas-upnp-org:service:" + ServiceName + ":1",
		EncodingSchema:       "http://schemas.xmlsoap.org/soap/encoding/",
		EnvelopeSchema:       "http://schemas.xmlsoap.org/soap/envelope/",
		ControlEndpoint:      serviceControlEndpoint,
		EventEndpoint:        serviceEventEndpoint,
		StateVariables:       s.StateVariables, // Pass all state variables for template access
		EventStateVariables:  eventStateVariables,
		Actions:              s.Actions,
		ServiceLowerName:     strings.ToLower(ServiceName),
		StateVariableStructs: stateVariableStructs.String(),
		ServiceStructFields:  serviceStructFields.String(),
	}

	// Create a new template and parse the template string.
	// Add the GetStateVariable function to the template's function map.
	tmpl, err := template.New("service").Funcs(template.FuncMap{
		"GetStateVariable": data.GetStateVariable,
	}).Parse(serviceTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing service template: %w", err)
	}

	// Execute the template with the data.
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("error executing service template: %w", err)
	}

	return buf.Bytes(), nil
}

const serviceTemplate = `
// Code generated by makeservice. DO NOT EDIT.

// Package {{.ServiceLowerName}} is a generated {{.ServiceName}} package.
package {{.ServiceLowerName}}

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	ServiceURN     = "{{.ServiceURN}}"
	EncodingSchema = "{{.EncodingSchema}}"
	EnvelopeSchema = "{{.EnvelopeSchema}}"
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

{{.StateVariableStructs}}

type Service struct {
	controlEndpoint *url.URL
	eventEndpoint   *url.URL

{{.ServiceStructFields}}

	location        *url.URL
	client          *http.Client
}

func NewService(opts ...ServiceOption) (*Service, error) {
	s := &Service{}

	c, err := url.Parse("{{.ControlEndpoint}}")
	if nil != err {
		return nil, fmt.Errorf("error parsing control endpoint: %w", err)
	}
	e, err := url.Parse("{{.EventEndpoint}}")
	if nil != err {
		return nil, fmt.Errorf("error parsing event endpoint: %w", err)
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.client == nil {
		return nil, errors.New("client is nil")
	}
	if s.location == nil {
		return nil, errors.New("location is nil")
	}

	s.controlEndpoint = s.location.ResolveReference(c)
	s.eventEndpoint = s.location.ResolveReference(e)

	return s, nil
}

func (s *Service) ControlEndpoint() *url.URL{
	return s.controlEndpoint
}

func (s *Service) EventEndpoint() *url.URL{
	return s.eventEndpoint
}

func (s *Service) Location() *url.URL{
	return s.location
}

func (s *Service) Client() *http.Client {
	return s.client
}

// internal use only
type envelope struct {
	XMLName xml.Name ` + "`xml:\"s:Envelope\"`" + `
	Xmlns string ` + "`xml:\"xmlns:s,attr\"`" + `
	EncodingStyle string ` + "`xml:\"s:encodingStyle,attr\"`" + `
	Body body ` + "`xml:\"s:Body\"`" + `
}

// internal use only
type body struct {
	XMLName xml.Name ` + "`xml:\"s:Body\"`" + `
{{range .Actions}}	{{.Name}} *{{.Name}}Args ` + "`xml:\"u:{{.Name}},omitempty\"`" + `
{{end}}}

// internal use only
type envelopeResponse struct {
	XMLName xml.Name ` + "`xml:\"Envelope\"`" + `
	Xmlns string ` + "`xml:\"xmlns:s,attr\"`" + `
	EncodingStyle string ` + "`xml:\"encodingStyle,attr\"`" + `
	Body bodyResponse ` + "`xml:\"Body\"`" + `
}

// internal use only
type bodyResponse struct {
	XMLName xml.Name ` + "`xml:\"Body\"`" + `
{{range .Actions}}	{{.Name}} *{{.Name}}Response ` + "`xml:\"{{.Name}}Response,omitempty\"`" + `
{{end}}}

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
	responseBody, err := ioutil.ReadAll(res.Body)
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

{{range .Actions}}
type {{.Name}}Args struct {
	Xmlns string ` + "`xml:\"xmlns:u,attr\"`" + `
{{range .Arguments}}{{if eq .Direction "in"}}
	{{$actionName := .Name}}
	{{$argName := .Name}}
	{{$relatedSvName := .RelatedStateVariable}}
	{{/* Use the custom GetStateVariable function passed to the template */}}
	{{$sv := call $.GetStateVariable $relatedSvName}}
	{{if not $sv}} {{/* This check is more for template safety, actual error handled in Go */}}
	// Error: StateVariable '{{$relatedSvName}}' not found for action '{{$actionName}}' argument '{{$argName}}'
	{{else}}
	{{if $sv.AllowedValueRange }}// Allowed Range: {{$sv.AllowedValueRange.Minimum}} -> {{$sv.AllowedValueRange.Maximum}} step: {{$sv.AllowedValueRange.Step}}
	{{end}}
	{{range $sv.AllowedValues}}// Allowed Value: {{.}}
	{{end}}
	{{.Name}} {{$sv.GoDataType}} ` + "`xml:\"{{.Name}}\"`" + `
	{{end}}
{{end}}{{end}}
}

type {{.Name}}Response struct {
{{range .Arguments}}{{if eq .Direction "out"}}
	{{$actionName := .Name}}
	{{$argName := .Name}}
	{{$relatedSvName := .RelatedStateVariable}}
	{{/* Use the custom GetStateVariable function passed to the template */}}
	{{$sv := call $.GetStateVariable $relatedSvName}}
	{{if not $sv}} // Error: StateVariable '{{$relatedSvName}}' not found for action '{{$actionName}}' argument '{{$argName}}'
	{{else}}
	{{.Name}} {{$sv.GoDataType}} ` + "`xml:\"{{.Name}}\"`" + `
	{{end}}
{{end}}{{end}}
}

func (s *Service) {{.Name}}(args *{{.Name}}Args) (*{{.Name}}Response, error) {
	args.Xmlns = ServiceURN
	r, err := s.exec("{{.Name}}",
		&envelope{
			EncodingStyle: EncodingSchema,
			Xmlns:         EnvelopeSchema,
			Body:          body{ {{.Name}}: args},
		})
	if err != nil { return nil, err }
	if r.Body.{{.Name}} == nil { return nil, fmt.Errorf("unexpected nil response body for {{.Name}} action") } // Propagate error

	return r.Body.{{.Name}}, nil
}
{{end}}

type UpnpEvent struct {
	XMLName xml.Name ` + "`xml:\"propertyset\"`" + `
	XMLNameSpace string ` + "`xml:\"xmlns:e,attr\"`" + `
	Properties []Property ` + "`xml:\"property\"`" + `
}

type Property struct {
	XMLName xml.Name ` + "`xml:\"property\"`" + `
{{range .EventStateVariables}}	{{.Name}} *{{.Name}} ` + "`xml:\"{{.Name}}\"`" + `
{{end}}}

func (zp *Service) ParseEvent(body []byte) ([]interface{}, error) {
	var evt UpnpEvent
	var events []interface{}
	err := xml.Unmarshal(body, &evt)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling event: %w", err)
	}
	for _, prop := range evt.Properties {
		_ = prop
		switch {
{{range .EventStateVariables}}		case prop.{{.Name}} != nil:
			events = append(events, *prop.{{.Name}})
{{end}}		}
	}
	return events, nil
}
`

func main() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage: makeservice <ServiceName> <ServiceControlEndpoint> <ServiceEventEndpoint> <ServiceXmlFile>")
		os.Exit(1)
	}
	serviceName := os.Args[1]
	serviceControlEndpoint := os.Args[2]
	serviceEventEndpoint := os.Args[3]
	serviceXml := os.Args[4]
	body, err := ioutil.ReadFile(serviceXml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading service XML file: %v\n", err)
		os.Exit(1)
	}
	dotgo, err := MakeServiceApi(serviceName, serviceControlEndpoint, serviceEventEndpoint, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making service API: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(dotgo))
}
