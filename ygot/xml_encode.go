package ygot

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"sort"

	"github.com/openconfig/ygot/util"
)

// xmlOutputConfig is used to determine how encodeXML should generate XML.
type xmlOutputConfig struct {
	// Namespace specifies the XML namespace to which the data structures being
	// converted to XML belong.
	Namespace string
	// SkipRootElement specifies whether the GoStruct supplied to EmitXML should
	// create the xml element.
	SkipRootElement bool
	// RootElement specifies the name of the root element.
	RootElement string
	// Namespace specifies the XML namespace to which the root data structure being
	// converted to XML belongs.
	RootNamespace string
	// Attrs specifies xml attributes that should be added to elements (using element name).
	Attrs map[string][]xml.Attr
}

// encodeXML renders the GoStruct s to xml string using a very simple set of
// conversion rules.
func encodeXML(s GoStruct, e *xml.Encoder, cfg xmlOutputConfig) error {
	var start xml.StartElement

	if !cfg.SkipRootElement {
		start.Name.Local = cfg.RootElement
		start.Name.Space = cfg.RootNamespace
	}

	err := xmlEncoder(e, s, start, cfg.Namespace, nil, cfg.Attrs)
	if err == nil {
		e.Flush()
	}

	return err
}

func xmlEncoder(
	e *xml.Encoder,
	obj interface{},
	start xml.StartElement,
	xmlns string,
	tags *reflect.StructTag,
	attrs map[string][]xml.Attr,
) error {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	// Dereference the object to get the actual value and type.
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}

		v = v.Elem()
		t = t.Elem()
	}

	switch k := v.Kind(); k {
	case reflect.Struct:
		if start.Name.Local != "" {
			addAttrs(&start, attrs)
			e.EncodeToken(start)
		}
		// Recurse into struct members.
		for i := 0; i < t.NumField(); i++ {
			tf := t.Field(i)
			vf := v.Field(i)

			// skip the fields that aren't exportable
			if util.IsSkippableField(tf) {
				continue
			}

			var s xml.StartElement
			s.Name.Local = tf.Tag.Get("path")
			s.Name.Space = xmlns

			if err := xmlEncoder(e, vf.Interface(), s, "", &tf.Tag, attrs); err != nil {
				return err
			}
		}
		if start.Name.Local != "" {
			e.EncodeToken(start.End())
		}

	case reflect.Map:
		var sortByOrder bool

		if tags != nil {
			if s := tags.Get("sort"); s == "user" {
				sortByOrder = true
			}
		}

		rkeys := v.MapKeys()
		if sortByOrder {
			// if sorted by insertion order we need to iterate over each
			// value and check that it has the appropriate index field, get its value,
			// and then sort the map by that value
			kmap := make(map[int]reflect.Value)
			indx := make([]int, len(rkeys))
			for i, k := range rkeys {
				ov := v.MapIndex(k)
				index, err := getOrderedMapIndex(ov)
				if err != nil {
					return err
				}
				if _, ok := kmap[index]; ok {
					return fmt.Errorf("duplicate 'OrderedBy User' index %v for list of %v", index, ov.Type())
				}
				kmap[index] = k
				indx[i] = index
			}
			sort.Ints(indx)

			for _, ind := range indx {
				xmlEncoder(e, v.MapIndex(kmap[ind]).Interface(), start.Copy(), "", tags, attrs)
			}

		} else {
			// otherwise we sort the map by key.
			kmap := make(map[string]reflect.Value)
			keys := make([]string, len(rkeys))
			for i, k := range rkeys {
				skey := k.String()
				kmap[skey] = k
				keys[i] = skey
			}
			sort.Strings(keys)

			for _, k := range keys {
				xmlEncoder(e, v.MapIndex(kmap[k]).Interface(), start.Copy(), "", tags, attrs)
			}
		}

	default:
		// for YANG enums we need to convert int values to their string representation
		if _, isEnum := obj.(GoEnum); isEnum {
			name, set, err := enumFieldToString(v, false)
			if err != nil {
				return fmt.Errorf("cannot resolve enumerated type, got err: %v", err)
			}
			if !set {
				break
			}
			obj = name
		}
		addAttrs(&start, attrs)
		e.EncodeElement(obj, start)
	}

	return nil
}

func addAttrs(start *xml.StartElement, m map[string][]xml.Attr) {
	if m == nil {
		return
	}

	if x, ok := m[start.Name.Local]; ok {
		start.Attr = append(start.Attr, x...)
	}
}

func getOrderedMapIndex(v reflect.Value) (int, error) {
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return 0, fmt.Errorf("type %s value is nil", v.Type())
		}

		v = v.Elem()
	}

	f := v.FieldByName("OrderedMapIndex")
	if !f.IsValid() {
		return 0, fmt.Errorf("type %s does not have a OrderedMapIndex function", v.Type())
	}

	return int(f.Int()), nil
}