package ygot

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"sort"

	"github.com/openconfig/gnmi/errlist"
)

// makeXML renders the GoStruct s to xml string using the conversion
// structXML rules (copied from structJSON).
func makeXML(s GoStruct, e *xml.Encoder, cfg xmlOutputConfig) error {
	v, err := structXML(s, "", cfg)
	if err != nil {
		return fmt.Errorf("XML conversion error: %v", err)
	}

	fmt.Printf("Produced: %+v\n", v)

	if err = e.Encode(v); err != nil {
		return fmt.Errorf("XML marshalling error: %v", err)
	}

	return nil
}

// structXML marshals a GoStruct to a map[string]interface{} which can be
// handed to XML marshal. parentMod specifies the module that the supplied
// GoStruct is defined within such that we are able to consider whether to
// set the xmlns namespace to an element. Returns an error if the GoStruct
// cannot be rendered to XML.
func structXML(s GoStruct, parentMod string, cfg xmlOutputConfig) (map[string]interface{}, error) {
	// TODO: remove json-specific code, replace jsonValue with xmlValue
	var errs errlist.List

	sval := reflect.ValueOf(s).Elem()
	stype := sval.Type()

	// Marshal into a map[string]interface{} which can be handed to
	// json.Marshal(Text)?
	jsonout := map[string]interface{}{}

	for i := 0; i < sval.NumField(); i++ {
		field := sval.Field(i)
		fType := stype.Field(i)

		// Module names to append to the path in RFC7951 output mode.
		var chMod string

		mapPaths, err := structTagToLibPaths(fType, newStringSliceGNMIPath([]string{}), false)
		if err != nil {
			errs.Add(fmt.Errorf("%s: %v", fType.Name, err))
			continue
		}

		// s is the fake root if its path tag is empty. In this case,
		// we want to forward the parent module to the child nodes.
		isFakeRoot := len(mapPaths) == 1 && mapPaths[0].Len() == 0
		if isFakeRoot {
			chMod = parentMod
		}

		value, err := xmlValue(field, chMod, cfg)
		if err != nil {
			errs.Add(err)
			continue
		}

		if value == nil {
			continue
		}

		if mp, ok := value.(map[string]interface{}); ok && len(mp) == 0 {
			continue
		}

		if isFakeRoot {
			if v, ok := value.(map[string]interface{}); ok {
				for mk, mv := range v {
					jsonout[mk] = mv
				}
			} else {
				errs.Add(fmt.Errorf("empty path specified for non-root entity"))
			}
			continue
		}

		for _, p := range mapPaths {
			parent := jsonout
			j := 0
			for ; j != p.Len()-1; j++ {
				k, err := p.StringElemAt(j)
				if err != nil {
					errs.Add(err)
					continue
				}

				if _, ok := parent[k]; !ok {
					parent[k] = map[string]interface{}{}
				}
				parent = parent[k].(map[string]interface{})
			}
			k, err := p.LastStringElem()
			if err != nil {
				errs.Add(err)
				continue
			}
			parent[k] = value
		}
	}

	if errs.Err() != nil {
		return nil, errs.Err()
	}

	return jsonout, nil
}

// xmlValue takes a reflect.Value which represents a struct field and
// constructs the representation that can be used to marshal the field to XML.
// The module within which the value is defined is specified by the parentMod string.
// Returns an error if one occurs during the mapping process.
func xmlValue(field reflect.Value, parentMod string, args xmlOutputConfig) (interface{}, error) {
	var value interface{}
	var errs errlist.List

	switch field.Kind() {
	case reflect.Map, reflect.Slice, reflect.Ptr, reflect.Interface:
		if field.IsNil() {
			return nil, nil
		}
	}

	appmod := false

	switch field.Kind() {
	case reflect.Map:
		var err error
		value, err = mapXML(field, parentMod, args)
		if err != nil {
			errs.Add(err)
		}
	case reflect.Ptr:
		switch field.Elem().Kind() {
		case reflect.Struct:
			goStruct, ok := field.Interface().(GoStruct)
			if !ok {
				return nil, fmt.Errorf("cannot map struct %v, invalid GoStruct", field)
			}

			var err error
			value, err = structXML(goStruct, parentMod, args)
			if err != nil {
				errs.Add(err)
			}
		default:
			value = field.Elem().Interface()
		}
		/*
			case reflect.Slice:

				isAnnotationSlice := func(v reflect.Value) bool {
					annoT := reflect.TypeOf((*Annotation)(nil)).Elem()
					return v.Type().Elem().Implements(annoT)
				}

				var err error
				switch {
				case isAnnotationSlice(field):
					value, err = jsonAnnotationSlice(field)
				default:
					value, err = jsonSlice(field, parentMod, args)
				}
				if err != nil {
					return nil, err
				}
		*/
	case reflect.Int64:
		// Enumerated values are represented as int64 in the generated Go structures.
		// For output, we map the enumerated value to the string name of the enum.
		v, set, err := enumFieldToString(field, appmod)
		if err != nil {
			return nil, err
		}

		// Skip if the enum has not been explicitly set in the schema.
		if !set {
			return nil, nil
		}
		value = v
		/*
			case reflect.Interface:
				// Union values that have more than one type are represented as a pointer to
				// an interface in the generated Go structures - extract the relevant value
				// and return this.
				var err error
				switch {
				case util.IsValueInterfaceToStructPtr(field):
					if value, err = unwrapUnionInterfaceValue(field, appmod); err != nil {
						return nil, err
					}
					if value != nil && reflect.TypeOf(value).Name() == BinaryTypeName {
						if value, err = jsonSlice(reflect.ValueOf(value), parentMod, args); err != nil {
							return nil, err
						}
						return value, nil
					}
				case field.Elem().Kind() == reflect.Slice && field.Elem().Type().Name() == BinaryTypeName:
					if value, err = jsonSlice(field.Elem(), parentMod, args); err != nil {
						return nil, err
					}
					return value, nil
				default:
					if value, err = resolveUnionVal(field.Elem().Interface(), appmod); err != nil {
						return nil, err
					}
				}
		*/
	case reflect.Bool:
		// A non-pointer field of type boolean is an empty leaf within the YANG schema.
		// For RFC7951 this is represented as a null JSON array (i.e., [null]). For internal
		// JSON if the leaf is present and set, it is rendered as 'true', or as nil otherwise.
		switch {
		case field.Bool():
			value = true
		}
	default:
		return nil, fmt.Errorf("got unexpected field type, was: %v", field.Kind())
	}

	if errs.Err() != nil {
		return nil, errs.Err()
	}
	return value, nil
}

// mapXML takes an input reflect.Value containing a map, and
// constructs the representation for XML marshalling that corresponds to it.
// The module within which the map is defined is specified by the parentMod
// argument.
func mapXML(field reflect.Value, parentMod string, args xmlOutputConfig) (interface{}, error) {
	var errs errlist.List
	mapKeyMap := map[string]reflect.Value{}
	// Order of elements determines the order in which keys will be processed.
	var mapKeys []string

	// YANG lists are marshalled into a JSON object array for IETF
	// JSON. We handle the keys in alphabetical order to ensure that
	// deterministic ordering is achieved in the output JSON.
	for _, k := range field.MapKeys() {
		keyval, err := keyValue(k, false)
		if err != nil {
			errs.Add(fmt.Errorf("invalid enumerated key: %v", err))
			continue
		}
		kn := fmt.Sprintf("%v", keyval)
		mapKeys = append(mapKeys, kn)
		mapKeyMap[kn] = k
	}

	sort.Strings(mapKeys)

	if len(mapKeys) == 0 {
		// empty list should be encoded as empty list
		return nil, nil
	}

	// Build the output that we expect. Since there is a difference between the IETF
	// and non-IETF forms, we simply choose vals to be interface{}, and then type assert
	// it later on. Since t cannot mutuate through this function we can guarantee that
	// the type assertions below will not cause panic, since we ensure that we know
	// what type of serialisation we're doing when we set the type.
	var vals interface{}
	vals = []interface{}{}
	for _, kn := range mapKeys {
		k := mapKeyMap[kn]
		goStruct, ok := field.MapIndex(k).Interface().(GoStruct)
		if !ok {
			errs.Add(fmt.Errorf("cannot map struct %v, invalid GoStruct", field))
			continue
		}

		val, err := structXML(goStruct, parentMod, args)
		if err != nil {
			errs.Add(err)
			continue
		}

		vals = append(vals.([]interface{}), val)
	}

	if errs.Err() != nil {
		return nil, errs.Err()
	}
	return vals, nil
}
