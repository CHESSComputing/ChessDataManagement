package main

import (
	"encoding/json"
	"testing"
)

// TestHelpersValidateData
func TestHelpersValidateData(t *testing.T) {
	initMetaDataService()

	data := `{"StringKey": "test","StrKeyMultipleValues": "foo,bla","ListKey": ["foo", "bla"],"FloatKey": 1.1,"BoolKey": true}`
	var record Record
	err := json.Unmarshal([]byte(data), &record)
	if err != nil {
		t.Errorf("Fail to unmarshal data record, error %v", err)
	}

	schema := fullPath("schemas/test.json")
	err = validateData(schema, record)
	if err != nil {
		t.Errorf("Fail validation of data record, error %v", err)
	}
}
