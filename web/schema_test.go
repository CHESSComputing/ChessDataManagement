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
  type: string
`
	tmpFile.Write([]byte(yamlData))
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// load json data
	fname := tmpFile.Name()
	s := Schema{}
	err = s.Load(fname)
	if err != nil {
		t.Fatal(err)
	}

	keys := s.Keys()
	fmt.Println("Schema keys", keys)
	okeys := s.OptionalKeys()
	fmt.Println("Schema optional keys", okeys)

	rec := make(Record)
	rec["pi"] = "person"
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
    {"key": "BeamEnergy", "type": "string", "optional": false}
]`
	tmpFile.Write([]byte(jsonData))
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// load json data
	fname := tmpFile.Name()
	s := Schema{}
	err = s.Load(fname)
	if err != nil {
		t.Fatal(err)
	}

	keys := s.Keys()
	fmt.Println("Schema keys", keys)
	okeys := s.OptionalKeys()
	fmt.Println("Schema optional keys", okeys)

	rec := make(Record)
	rec["pi"] = "person"
	err = s.Validate(rec)
	if err != nil {
		t.Fatal(err)
	}
}
