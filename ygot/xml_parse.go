package ygot

import (
	"bytes"
	"encoding/xml"
)

// ParseXMLConfig specifies how the XML should be parse by the ParseXML function.
type ParseXMLConfig struct {
	// Namespace specifies the XML namespace to which the data structures being
	// converted from XML belong.
	Namespace string
	// SkipRootStruct specifies whether the GoStruct supplied to decodeXML should
	// be skipped when parsing the data. Useful for top-level Device stuct which is
	// not present in the parsed XML but is used as a top-level parent of the object tree.
	// e.g. with the following object relationship:
	// type Device struct {
	//   Configuration *Configuration ` path:"configuration"`
	//   Other *Other                 ` path:"other"`
	// }
	// when SkipRootStruct=true:
	//   decodeXML(&Device{}, ..) will succeed when given xml with <configuration />
	//   decodeXML(&Device{}, ..) will fail when given xml with <device />
	// and SkipRootStruct=false:
	//   decodeXML(&Device{}, ..) will succeed when given xml with <device />
	//   decodeXML(&Device{}, ..) will fail when given xml with <configuration />
	SkipRootStruct bool
	// RootElement specifies the name of the root element (if SkipRootStruct == false).
	RootElement string
}

// ParseXML takes an input xml and produces ValidatedGoStruct from it.
func ParseXML(xmlData []byte, s ValidatedGoStruct, opts *ParseXMLConfig) error {
	cfg := xmlInputConfig{
		SkipRootStruct: true,
	}

	if opts != nil {
		cfg.Namespace = opts.Namespace
		cfg.SkipRootStruct = opts.SkipRootStruct
		cfg.RootElement = opts.RootElement
	}

	dec := xml.NewDecoder(bytes.NewReader(xmlData))
	return decodeXML(s, dec, cfg)
}
