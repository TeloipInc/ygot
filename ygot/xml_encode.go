package ygot

import (
	"encoding/xml"
	"reflect"
)

// encodeXML renders the GoStruct s to xml string using a very simple set of
// conversion rules.
func encodeXML(s GoStruct, e *xml.Encoder, cfg xmlOutputConfig) error {
	var start xml.StartElement

	if !cfg.SkipRootElement {
		start.Name.Local = cfg.RootElement
		start.Name.Space = cfg.RootNamespace
	}

	err := xmlEncoder(e, s, start, cfg.Namespace)
	if err == nil {
		e.Flush()
	}

	return err
}

func xmlEncoder(e *xml.Encoder, obj interface{}, start xml.StartElement, xmlns string) error {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	// Dereference the object to get the actual value and type.
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
		t = t.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		if start.Name.Local != "" {
			e.EncodeToken(start)
		}
		// Recurse into struct members.
		for i := 0; i < t.NumField(); i++ {
			tf := t.Field(i)
			vf := v.Field(i)

			var s xml.StartElement
			s.Name.Local = tf.Tag.Get("path")
			s.Name.Space = xmlns

			if err := xmlEncoder(e, vf.Interface(), s, ""); err != nil {
				return err
			}
		}
		if start.Name.Local != "" {
			e.EncodeToken(start.End())
		}

	case reflect.Map:
		// Iterate the map's values, using the same start element for each.
		iter := v.MapRange()
		for iter.Next() {
			xmlEncoder(e, iter.Value().Interface(), start, "")
		}

	default:
		e.EncodeElement(obj, start)
	}

	return nil
}
