package ygot

import (
	"encoding/xml"
	"fmt"
	"strings"
)

const (
	indentPrefix string = ""
	rootElement  string = "root"
)

// XMLEncode is an enumerated integer value indicating how the xml is encoded.
type XMLEncode int

const (
	// EncodeXML forces the use of encodeXML function to encode XML.
	EncodeXML XMLEncode = iota
	// MakeXML forces the use of makeXML function to encode XML.
	MakeXML
)

// EmitxmlConfig specifies how the XML should be created by the EmitXML function.
type EmitXMLConfig struct {
	// Encode specifies how the XML is produced by ygot package. By default,
	// EncodeXML is chosen.
	Encode XMLEncode
	// Indent is the string used for indentation within the XML output. The
	// default value is three spaces.
	Indent string
	// IndentPrefix is the string used for prefixing indentation within the XML output.
	// The default value is empty string.
	IndentPrefix string
	// Namespace specifies the XML namespace to which the data structures being
	// converted to XML belong.
	Namespace string
	// RootElement specifies the name of the root element.
	// The default value is "root".
	RootElement string
	// SkipValidation specifies whether the GoStruct supplied to EmitXML should
	// be validated before emitting its content. Validation is skipped when it
	// is set to true.
	SkipValidation bool
	// ValidationOpts is the set of options that should be used to determine how
	// the schema should be validated. This allows fine-grained control of particular
	// validation rules in the case that a partially populated data instance is
	// to be emitted.
	ValidationOpts []ValidationOption
}

// xmlOutputConfig is used to determine how makeXML/encodeXML should generate XML.
type xmlOutputConfig struct {
	// Namespace specifies the XML namespace to which the data structures being
	// converted to XML belong.
	Namespace string
	// RootElement specifies the name of the root element.
	RootElement string
}

// EmitXML takes an input ValidatedGoStruct (produced by ygen with validation enabled)
// and serialises it to a XML string.
func EmitXML(s ValidatedGoStruct, opts *EmitXMLConfig) (string, error) {
	var (
		vopts          []ValidationOption
		skipValidation bool
	)

	cfg := xmlOutputConfig{
		RootElement: rootElement,
	}

	encode := EncodeXML
	if opts != nil {
		vopts = opts.ValidationOpts
		skipValidation = opts.SkipValidation
		encode = opts.Encode

		cfg.Namespace = opts.Namespace

		if opts.RootElement != "" {
			cfg.RootElement = opts.RootElement
		}
	}

	if !skipValidation {
		if err := s.Validate(vopts...); err != nil {
			return "", fmt.Errorf("validation err: %v", err)
		}
	}

	sb := &strings.Builder{}
	enc := xml.NewEncoder(sb)
	indent := indentString
	prefix := indentPrefix
	if opts != nil {
		if opts.Indent != "" {
			indent = opts.Indent
		}
		if opts.IndentPrefix != "" {
			prefix = opts.IndentPrefix
		}
	}
	enc.Indent(prefix, indent)

	if encode == MakeXML {
		err := makeXML(s, enc, cfg)
		if err != nil {
			return "", fmt.Errorf("makeXML error: %v", err)
		}
	} else {
		err := encodeXML(s, enc, cfg)
		if err != nil {
			return "", fmt.Errorf("encodeXML error: %v", err)
		}
	}

	return sb.String(), nil
}
