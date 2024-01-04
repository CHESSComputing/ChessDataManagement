package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestFilesDB
func TestFilesDB(t *testing.T) {
	initMetaDataService()
	// initialize FilesDB connection
	var err error
	FilesDB, err = InitFilesDB()
	defer FilesDB.Close()
	if err != nil {
		log.Printf("FilesDB error: %v\n", err)
	}

	// prepare our data for insertion
	did := "123"
	cycle := "cycle"
	beamline := "beamline"
	btr := "btr"
	sample := "sample"
	dataset := fmt.Sprintf("/%s/%s/%s/%s", cycle, beamline, btr, sample)
	path := "/tmp"
	path = filepath.Join("/tmp", os.Getenv("USER")) // for testing purposes
	files := FindFiles(path)
	if len(files) == 0 {
		t.Errorf("Unable to find any files in directory=%s", path)
	}

	err = InsertFiles(did, dataset, path)
	if err != nil {
		t.Fatal(err)
	}
	files, err = getFiles(did)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Errorf("No files found in database for did=%v", did)
	}

	// insert another dataset with different cycle
	did = "456"
	cycle = "cycle-2"
	dataset = fmt.Sprintf("/%s/%s/%s/%s", cycle, beamline, btr, sample)
	path = filepath.Join("/tmp", os.Getenv("USER")) // for testing purposes
	err = InsertFiles(did, dataset, path)
	if err != nil {
		t.Fatal(err)
	}

	// check if btr/beamline/sample tables do not expand
	samples, err := getSamples()
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 1 {
		t.Fatal("wrong number of samples")
	}
	btrs, err := getBtrs()
	if err != nil {
		t.Fatal(err)
	}
	if len(btrs) != 1 {
		t.Fatal("wrong number of btrs")
	}
	beamlines, err := getBeamlines()
	if err != nil {
		t.Fatal(err)
	}
	if len(beamlines) != 1 {
		t.Fatal("wrong number of beamlines")
	}

	// get list of datasets
	dsets, err := getDatasets()
	if err != nil {
		t.Fatal(err)
	}
	if len(dsets) != 2 {
		t.Fatal("wrong number of datasets")
	}

}
