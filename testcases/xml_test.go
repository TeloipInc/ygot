package testcases

import (
	"encoding/xml"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/openconfig/ygot/ygot"
)

// generate code from following models, to be used by test cases below.
//go:generate go run ../generator/generator.go -path=yang -output_file=generated_test_rpc.go -package_name=testcases -generate_fakeroot -fakeroot_name=device -compress_paths=false -shorten_enum_leaf_names -typedef_enum_with_defmod -exclude_modules=ietf-interfaces ../testdata/modules/test-rpc.yang

// TestEmitXML tests how ygot.EmitXML generates xml from data structures
func TestEmitXML(t *testing.T) {
	d := &Device{
		Configuration: &TestRpc_Configuration{
			Parent: &TestRpc_Configuration_Parent{
				GroupLeaf: ygot.String("group data"),
				EnumField: TestRpc_YType_a,
			},
		},
	}
	d.Configuration.Parent.NewChildren("child1")
	d.Configuration.Parent.NewChildren("child2")

	xml, err := ygot.EmitXML(d, &ygot.EmitXMLConfig{
		Indent:          "  ",
		IndentPrefix:    "",
		Namespace:       "urn:r",
		SkipRootElement: true,
		SkipValidation:  false,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedXml := `<configuration xmlns="urn:r">
  <parent>
    <children>
      <name>child1</name>
    </children>
    <children>
      <name>child2</name>
    </children>
    <enum-field>a</enum-field>
    <group-leaf>group data</group-leaf>
  </parent>
</configuration>`

	if xml != expectedXml {
		t.Fatalf("wanted: %v\ngot: %v\n", expectedXml, xml)
	}
}

// TestParseXML0 tests how ygot.ParseXML parses an object with no data.
func TestParseXML0(t *testing.T) {
	xmlToParse := `<parent xmlns="urn:r">
</parent>`

	var p TestRpc_Configuration_Parent
	err := ygot.ParseXML([]byte(xmlToParse), &p, &ygot.ParseXMLConfig{
		Namespace:      "urn:r",
		SkipRootStruct: false,
		RootElement:    "parent",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := TestRpc_Configuration_Parent{}
	if !reflect.DeepEqual(&ref, &p) {
		t.Fatalf("wanted: %v\ngot: %v\n", spew.Sdump(ref), spew.Sdump(p))
	}
}

// TestParseXML1 tests how ygot.ParseXML parses simple fields.
func TestParseXML1(t *testing.T) {
	xmlToParse := `<parent xmlns="urn:r">
  <group-leaf>group data</group-leaf>
</parent>`

	var p TestRpc_Configuration_Parent
	err := ygot.ParseXML([]byte(xmlToParse), &p, &ygot.ParseXMLConfig{
		Namespace:      "urn:r",
		SkipRootStruct: false,
		RootElement:    "parent",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := TestRpc_Configuration_Parent{
		GroupLeaf: ygot.String("group data"),
	}

	if !reflect.DeepEqual(&ref, &p) {
		t.Fatalf("wanted: %v\ngot: %v\n", spew.Sdump(ref), spew.Sdump(p))
	}
}

// TestParseXML2 tests how ygot.ParseXML parses enums.
func TestParseXML2(t *testing.T) {
	xmlToParse := `<parent xmlns="urn:r">
  <enum-field>a</enum-field>
</parent>`

	var p TestRpc_Configuration_Parent
	err := ygot.ParseXML([]byte(xmlToParse), &p, &ygot.ParseXMLConfig{
		Namespace:      "urn:r",
		SkipRootStruct: false,
		RootElement:    "parent",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := TestRpc_Configuration_Parent{
		EnumField: TestRpc_YType_a,
	}

	if !reflect.DeepEqual(&ref, &p) {
		t.Fatalf("wanted: %v\ngot: %v\n", spew.Sdump(ref), spew.Sdump(p))
	}
}

// TestParseXML3 tests how ygot.ParseXML parses maps.
func TestParseXML3(t *testing.T) {
	xmlToParse := `<parent xmlns="urn:r">
  <children>
    <name>child1</name>
  </children>
  <children>
    <name>child2</name>
  </children>
</parent>`

	var p TestRpc_Configuration_Parent
	err := ygot.ParseXML([]byte(xmlToParse), &p, &ygot.ParseXMLConfig{
		Namespace:      "urn:r",
		SkipRootStruct: false,
		RootElement:    "parent",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := TestRpc_Configuration_Parent{}
	ref.NewChildren("child1")
	ref.NewChildren("child2")

	if !reflect.DeepEqual(&ref, &p) {
		t.Fatalf("wanted: %v\ngot: %v\n", spew.Sdump(ref), spew.Sdump(p))
	}
}

// TestParseXML4 tests how ygot.ParseXML parses the xml into data structures.
// SkipRootStruct = true should prevent Device from being searched in the xml.
func TestParseXML4(t *testing.T) {
	xmlToParse := `<configuration xmlns="urn:r">
  <parent>
    <children>
      <name>child1</name>
    </children>
    <children>
      <name>child2</name>
    </children>
    <enum-field>a</enum-field>
    <group-leaf>group data</group-leaf>
  </parent>
</configuration>`

	var d Device
	err := ygot.ParseXML([]byte(xmlToParse), &d, &ygot.ParseXMLConfig{
		Namespace:      "urn:r",
		SkipRootStruct: true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := Device{
		Configuration: &TestRpc_Configuration{
			Parent: &TestRpc_Configuration_Parent{
				GroupLeaf: ygot.String("group data"),
				EnumField: TestRpc_YType_a,
			},
		},
	}
	d.Configuration.Parent.NewChildren("child1")
	d.Configuration.Parent.NewChildren("child2")

	if !reflect.DeepEqual(&ref, &d) {
		t.Fatalf("wanted: %v\ngot: %v\n", spew.Sdump(ref), spew.Sdump(d))
	}
}

// TestParseXML5 tests how ygot.ParseXML parses the xml into data structures
// SkipRootStruct = false will match TestRpc_Configuration with the top-level 'configuration' element.
func TestParseXML5(t *testing.T) {
	xmlToParse := `<configuration xmlns="urn:r">
  <parent>
    <children>
      <name>child1</name>
    </children>
    <children>
      <name>child2</name>
    </children>
    <enum-field>a</enum-field>
    <group-leaf>group data</group-leaf>
  </parent>
</configuration>`

	var c TestRpc_Configuration
	err := ygot.ParseXML([]byte(xmlToParse), &c, &ygot.ParseXMLConfig{
		Namespace:      "urn:r",
		SkipRootStruct: false,
		RootElement:    "configuration",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := TestRpc_Configuration{
		Parent: &TestRpc_Configuration_Parent{
			GroupLeaf: ygot.String("group data"),
			EnumField: TestRpc_YType_a,
		},
	}
	ref.Parent.NewChildren("child1")
	ref.Parent.NewChildren("child2")

	if !reflect.DeepEqual(&ref, &c) {
		t.Fatalf("wanted: %v\ngot: %v\n", spew.Sdump(ref), spew.Sdump(c))
	}
}

// TestUnmarshal runs the stock xml.Unmarshal.
// This test is designed to fail, as the stock Unmarshaller doesn't know how to
// deal with any of the fields, since they are lacking 'xml' tags.
func TestUnmarshal(t *testing.T) {
	xmlToParse := `<configuration xmlns="urn:r">
  <parent>
    <children>
      <name>child1</name>
    </children>
    <children>
      <name>child2</name>
    </children>
    <enum-field>a</enum-field>
    <group-leaf>group data</group-leaf>
  </parent>
</configuration>`

	var c TestRpc_Configuration
	err := xml.Unmarshal([]byte(xmlToParse), &c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := TestRpc_Configuration{
		Parent: &TestRpc_Configuration_Parent{
			GroupLeaf: ygot.String("group data"),
			EnumField: TestRpc_YType_a,
		},
	}
	ref.Parent.NewChildren("child1")
	ref.Parent.NewChildren("child2")

	if reflect.DeepEqual(&ref, &c) {
		t.Fatalf("this shouldn't be possible!")
	}
}
