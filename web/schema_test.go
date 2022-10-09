package main

import (
	"fmt"
	"os"
	"testing"
)

// TestSchemaYaml tests schema yaml file
func TestSchemaYaml(t *testing.T) {
	tmpFile, err := os.CreateTemp(os.TempDir(), "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	yamlData := `
- key: Pi
  optional: true
  type: string
- key: BeamEnergy
  optional: false
  type: int
`
	tmpFile.Write([]byte(yamlData))
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// load json data
	fname := tmpFile.Name()
	s := &Schema{FileName: fname}
	err = s.Load()
	if err != nil {
		t.Fatal(err)
	}

	keys, err := s.Keys()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Schema keys", keys)
	okeys, err := s.OptionalKeys()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Schema optional keys", okeys)

	rec := make(Record)
	rec["Pi"] = "person"
	rec["BeamEnergy"] = 123
	err = s.Validate(rec)
	if err != nil {
		t.Fatal(err)
	}
}

// TestSchemaJson tests schema json file
func TestSchemaJson(t *testing.T) {
	tmpFile, err := os.CreateTemp(os.TempDir(), "*.json")
	if err != nil {
		t.Fatal(err)
	}
	jsonData := `[
    {"key": "Pi", "type": "string", "optional": true},
    {"key": "BeamEnergy", "type": "int", "optional": false}
]`
	tmpFile.Write([]byte(jsonData))
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// load json data
	fname := tmpFile.Name()
	s := &Schema{FileName: fname}
	err = s.Load()
	if err != nil {
		t.Fatal(err)
	}

	keys, err := s.Keys()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Schema keys", keys)
	okeys, err := s.OptionalKeys()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Schema optional keys", okeys)

	rec := make(Record)
	rec["Pi"] = "person"
	rec["BeamEnergy"] = 123
	err = s.Validate(rec)
	if err != nil {
		t.Fatal(err)
	}
}
