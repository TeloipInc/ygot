package ygot

import (
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/openconfig/ygot/util"
)

// xmlInputConfig is used to determine how decodeXML should process XML.
type xmlInputConfig struct {
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
	// when SkipRootStruct:
	//   decodeXML(&Device{}, ..) will succeed when given xml with <configuration />
	// and if !SkipRootStruct:
	//   decodeXML(&Device{}, ..) will succeed when given xml with <device />
	SkipRootStruct bool
	// RootElement specifies the name of the root element (if SkipRootStruct == false).
	RootElement string
	//
	// xmlDecoder parameters
	//
	objElementName  string
	objElementEnded bool
	objElementData  bool
	objField        *reflect.StructField
	objValue        *reflect.Value
}

// decodeXML decodes the xml into specified GoStruct.
func decodeXML(s GoStruct, d *xml.Decoder, cfg xmlInputConfig) error {
	st := reflect.TypeOf(s)

	if st.Kind() == reflect.Ptr || st.Kind() == reflect.Interface {
		st = st.Elem()
	}

	// move the xml offset to RootElement's children
	if !cfg.SkipRootStruct {
		cfg.objElementName = cfg.RootElement

		_, err := nextStartElement(d, cfg.objElementName)
		if err != nil {
			return fmt.Errorf("unexpected error when looking for <%s>: %v", cfg.objElementName, err)
		}

		fmt.Printf("<%s>\n", cfg.objElementName)
	}

	// the top-level object is not expecting data, only child elements
	cfg.objElementData = false

	// parse s's fields
	err := xmlDecoder(d, s, &cfg)
	if err != nil {
		return fmt.Errorf("decodeXML: %v", err)
	}

	// ensure the RootElement is properly terminated
	if !cfg.SkipRootStruct && !cfg.objElementEnded {
		_, err := nextEndElement(d, cfg.objElementName)
		if err != nil {
			return fmt.Errorf("unexpected error when looking for </%s>: %v", cfg.objElementName, err)
		}
	}

	return nil
}

// xmlDecoder attempts to parse the child fields of the obj from the xml,
// that means that the decoder's offset should already be positioned to
// point at the first child/field start element.
//
// xmlDecoder sets objElementEnded IF it read the end element that closes
// the xmlDecoder caller's start element.
func xmlDecoder(
	d *xml.Decoder,
	obj interface{},
	objCfg *xmlInputConfig,
) error {
	for {
		// read the token, ideally this will be the objElement's child start element
		t, err := d.Token()
		if err != nil {
			// TODO: EOF?
			return err
		}

		switch x := t.(type) {
		case xml.StartElement:
			name := x.Name.Local

			sf, fv := findStructField(obj, name)
			if sf == nil {
				return fmt.Errorf("unexpected <%s> when handling object '%s', was looking for a field element", name, objCfg.objElementName)
			}

			fmt.Printf("<%s>\n", name)
			fmt.Printf("field: %s %s\n", sf.Name, sf.Type.String())

			childCfg := *objCfg
			childCfg.objElementName = name
			childCfg.objElementEnded = false

			if sf.Type.Kind() == reflect.Pointer {
				// assign new instance the field value, but only if it's not set yet (e.g. maps)
				if fv.IsNil() {
					fv.Set(reflect.New(sf.Type.Elem()))
				}
			}
			// else TODO: map new?

			// figure out whether we need to read the element's data or child elements
			ty := sf.Type
			if ty.Kind() == reflect.Pointer || ty.Kind() == reflect.Interface {
				ty = ty.Elem()
			}
			switch ty.Kind() {
			// TODO: maps
			case reflect.Struct:
				childCfg.objElementData = false
				childCfg.objField = nil
				childCfg.objValue = nil
			default:
				childCfg.objElementData = true
				childCfg.objField = sf
				childCfg.objValue = fv
			}

			err := xmlDecoder(d, fv.Interface(), &childCfg)
			if err != nil {
				// TODO: nest error
				return err
			}

		case xml.EndElement:
			if x.Name.Local == objCfg.objElementName {
				fmt.Printf("</%s>\n", objCfg.objElementName)
				objCfg.objElementEnded = true
				return nil
			}
			return fmt.Errorf("unexpected end element </%s> when handling object '%s'", x.Name.Local, objCfg.objElementName)

		case xml.CharData:
			trimmed := strings.TrimSpace(string(x))
			if len(trimmed) != 0 {
				if !objCfg.objElementData {
					return fmt.Errorf("unexpected chardata '%s' when handling object '%s'", trimmed, objCfg.objElementName)
				}

				// store current element's value
				err := unmarshalData(trimmed, objCfg)
				if err != nil {
					return err
				}
			}
		}
	}
}

func unmarshalData(data string, objCfg *xmlInputConfig) error {
	dst := *objCfg.objValue
	dst0 := dst

	if dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}

	// handle yang enums first
	if e, isEnum := dst.Interface().(GoEnum); isEnum {
		lookup, ok := e.ΛMap()[dst.Type().Name()]
		if !ok {
			return fmt.Errorf("cannot map enumerated value as type %s was unknown", dst.Type().Name())
		}

		for eval, edef := range lookup {
			if edef.Name == data {
				dst.SetInt(eval)
				return nil
			}
		}
	}

	// unmarshal primitive data
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if len(data) == 0 {
			dst.SetInt(0)
			return nil
		}
		itmp, err := strconv.ParseInt(strings.TrimSpace(data), 10, dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetInt(itmp)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if len(data) == 0 {
			dst.SetUint(0)
			return nil
		}
		utmp, err := strconv.ParseUint(strings.TrimSpace(data), 10, dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetUint(utmp)
	case reflect.Float32, reflect.Float64:
		if len(data) == 0 {
			dst.SetFloat(0)
			return nil
		}
		ftmp, err := strconv.ParseFloat(strings.TrimSpace(data), dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetFloat(ftmp)
	case reflect.Bool:
		if len(data) == 0 {
			dst.SetBool(false)
			return nil
		}
		value, err := strconv.ParseBool(strings.TrimSpace(data))
		if err != nil {
			return err
		}
		dst.SetBool(value)
	case reflect.String: // not possible with yang generator
		dst.SetString(data)
	case reflect.Slice: // not sure if possible with yang generator
		if len(data) == 0 {
			// non-nil to flag presence
			data = string([]byte{})
		}
		dst.SetBytes([]byte(data))
	default:
		return errors.New("cannot unmarshal into " + dst0.Type().String())
	}
	return nil
}

func nextStartElement(d *xml.Decoder, n string) (*xml.StartElement, error) {
	t, err := d.Token()
	if err != nil {
		return nil, err
	}

	switch x := t.(type) {
	case xml.StartElement:
		if x.Name.Local != n {
			return nil, fmt.Errorf("unexpected start element <%s> when looking for <%s>", x.Name.Local, n)
		}
		return &x, nil
	case xml.EndElement:
		return nil, fmt.Errorf("unexpected end element </%s> when looking for <%s>", x.Name.Local, n)
	case xml.CharData:
		return nil, fmt.Errorf("unexpected data '%v' when looking for <%s>", x, n)
	case xml.Comment:
		return nil, fmt.Errorf("unexpected comment '%v' when looking for <%s>", x, n)
	case xml.ProcInst:
		return nil, fmt.Errorf("unexpected proc inst '%v' when looking for <%s>", x, n)
	case xml.Directive:
		return nil, fmt.Errorf("unexpected directive '%v' when looking for <%s>", x, n)
	default:
		return nil, fmt.Errorf("unknown xml token '%v' when looking for <%s>", x, n)
	}
}

func nextEndElement(d *xml.Decoder, n string) (*xml.EndElement, error) {
	t, err := d.Token()
	if err != nil {
		return nil, err
	}

	switch x := t.(type) {
	case xml.StartElement:
		return nil, fmt.Errorf("unexpected start element <%s> when looking for </%s>", x.Name.Local, n)
	case xml.EndElement:
		if x.Name.Local != n {
			return nil, fmt.Errorf("unexpected end element </%s> when looking for </%s>", x.Name.Local, n)
		}
		return &x, nil
	case xml.CharData:
		return nil, fmt.Errorf("unexpected data '%v' when looking for </%s>", x, n)
	case xml.Comment:
		return nil, fmt.Errorf("unexpected comment '%v' when looking for </%s>", x, n)
	case xml.ProcInst:
		return nil, fmt.Errorf("unexpected proc inst '%v' when looking for </%s>", x, n)
	case xml.Directive:
		return nil, fmt.Errorf("unexpected directive '%v' when looking for </%s>", x, n)
	default:
		return nil, fmt.Errorf("unknown xml token '%v' when looking for </%s>", x, n)
	}
}

func findStructField(obj interface{}, name string) (*reflect.StructField, *reflect.Value) {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	// Dereference the object to get the actual value and type.
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, nil
		}

		v = v.Elem()
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		fv := v.Field(i)

		// skip the fields that aren't exportable
		if util.IsSkippableField(ft) {
			continue
		}

		path := ft.Tag.Get("path")
		if path == name {
			return &ft, &fv
		}
	}

	return nil, nil
}
