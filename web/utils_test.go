package main

import (
	"log"
	"testing"
)

// helper function to initialize MetaData service
func initMetaDataService() {
	// initialize schema manager which holds our schemas
	configFile := "server_test.json"
	ParseConfig(configFile)
	// use verbose log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// initialize schema manager
	initSchemaManager()

	// init MongoDB
	InitMongoDB(Config.URI)
}

// helper function to initialize SchemaManager
func initSchemaManager() {
	// initialize schema manager
	_smgr = SchemaManager{}
	for _, fname := range Config.SchemaFiles {
		fname = fullPath(fname)
		_, err := _smgr.Load(fname)
		if err != nil {
			log.Fatalf("unable to load %s error %v", fname, err)
		}
		_beamlines = append(_beamlines, fileName(fname))
	}
}

// TestUtilsInList
func TestUtilsInList(t *testing.T) {
	vals := []string{"1", "2", "3"}
	res := InList("1", vals)
	if res == false {
		t.Error("Fail TestInList")
	}
	res = InList("5", vals)
	if res == true {
		t.Error("Fail TestInList")
	}
}

// TestUtilsSet
func TestUtilsSet(t *testing.T) {
	vals := []string{"a", "b", "c", "a"}
	res := List2Set(vals)
	if len(res) != 3 {
		t.Error("Fail TestUtilsSet")
	}
}
