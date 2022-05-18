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

// EmitxmlConfig specifies how the XML should be created by the EmitXML function.
type EmitXMLConfig struct {
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
	// Namespace specifies the XML namespace to which the root data structure being
	// converted to XML belongs.
	RootNamespace string
	// SkipRootElement specifies whether the GoStruct supplied to EmitXML should
	// create the xml element.
	SkipRootElement bool
	// SkipValidation specifies whether the GoStruct supplied to EmitXML should
	// be validated before emitting its content. Validation is skipped when it
	// is set to true.
	SkipValidation bool
	// ValidationOpts is the set of options that should be used to determine how
	// the schema should be validated. This allows fine-grained control of particular
	// validation rules in the case that a partially populated data instance is
	// to be emitted.
	ValidationOpts []ValidationOption
	// Attrs specifies xml attributes that should be added to elements (using element name).
	Attrs map[string][]xml.Attr
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

	if opts != nil {
		vopts = opts.ValidationOpts
		skipValidation = opts.SkipValidation

		cfg.Namespace = opts.Namespace

		if opts.RootElement != "" {
			cfg.RootElement = opts.RootElement
		}

		if opts.RootNamespace != "" {
			cfg.RootNamespace = opts.RootNamespace
		}

		cfg.SkipRootElement = opts.SkipRootElement

		cfg.Attrs = opts.Attrs
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

	err := encodeXML(s, enc, cfg)
	if err != nil {
		return "", fmt.Errorf("encodeXML error: %v", err)
	}

	return sb.String(), nil
}
