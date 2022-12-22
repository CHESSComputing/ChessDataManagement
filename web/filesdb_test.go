package main

import (
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
	did := int64(123)
	experiment := "CHESS"
	processing := "processing"
	tier := "tier"
	path := "/tmp"
	if os.Getenv("USER") != "runner" { // github action user
		path = filepath.Join("/tmp", os.Getenv("USER")) // for testing purposes
	}
	files := FindFiles(path)
	if len(files) == 0 {
		t.Errorf("Unable to find any files in %s", path)
	}

	err = InsertFiles(did, experiment, processing, tier, path)
	if err != nil {
		t.Error(err)
	}
	files, err = getFiles(did)
	if err != nil {
		t.Error(err)
	}
	if len(files) == 0 {
		t.Errorf("No files found in %s", path)
	}
}
