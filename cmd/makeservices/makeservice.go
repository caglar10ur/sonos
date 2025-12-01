package main

import (
	"bytes"
	_ "embed"
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

func (s *StateVariable) GoDataType() string {
	if len(s.AllowedValues) > 0 {
		return strings.ReplaceAll(s.Name, "A_ARG_TYPE_", "") + "Enum"
	}
	return s.UnderlyingGoType()
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

//go:embed service.tmpl
var serviceTemplate string

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
		"sanitize": func(s string) string {
			s = strings.ReplaceAll(s, "-", "_")
			s = strings.ReplaceAll(s, ".", "_")
			s = strings.ReplaceAll(s, ":", "_")
			s = strings.ReplaceAll(s, " ", "_")
			s = strings.ReplaceAll(s, "A_ARG_TYPE_", "")
			return s
		},
	}).Parse(serviceTemplate)
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
