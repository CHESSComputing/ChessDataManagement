package main

import (
	"encoding/json"
	"testing"
)

// TestHelpersValidateData
func TestHelpersValidateData(t *testing.T) {
	initMetaDataService()

	data := `{"StringKey": "test","StrKeyMultipleValues": "3A,3B","ListKey": ["3A", "3B"],"FloatKey": 1.1,"BoolKey": true}`
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

// TestHelpersValidateDataWrong
func TestHelpersValidateDataWrong(t *testing.T) {
	initMetaDataService()

	// our test.json schema only contains real beamline values, e.g. 3A, 3B, therefore
	// if we pass wrong value we should catch this error.
	// Here we provide one correct value (3A) and one wrong value (foo)
	data := `{"StringKey": "test","StrKeyMultipleValues": "3A,foo","ListKey": ["3A", "foo"],"FloatKey": 1.1,"BoolKey": true}`
	var record Record
	err := json.Unmarshal([]byte(data), &record)
	if err != nil {
		t.Errorf("Fail to unmarshal data record, error %v", err)
	}

	schema := fullPath("schemas/test.json")
	err = validateData(schema, record)
	// validation MUST fail here due to wrong "foo" value in multiple array
	if err == nil {
		t.Errorf("Fail validation of data record, error %v", err)
	}
}
