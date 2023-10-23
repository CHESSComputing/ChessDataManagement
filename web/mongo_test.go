package main

import (
	"testing"

	bson "go.mongodb.org/mongo-driver/bson"
)

// TestMongoInsert
func TestMongoInsert(t *testing.T) {
	// our db attributes
	dbname := "chess"
	collname := "test"
	InitMongoDB(Config.URI)

	// remove all records in test collection
	Remove(dbname, collname, bson.M{})

	// insert one record
	var records []Record
	dataset := "/a/b/c"
	rec := Record{"dataset": dataset}
	records = append(records, rec)
	Insert(dbname, collname, records)

	// look-up one record
	spec := bson.M{"dataset": dataset}
	idx := 0
	limit := 1
	records = MongoGet(dbname, collname, spec, idx, limit)
	if len(records) != 1 {
		t.Errorf("unable to find records using spec '%s', records %+v", spec, records)
	}

	// modify our record
	rec = Record{"dataset": dataset, "test": 1}
	records = []Record{}
	records = append(records, rec)
	err := MongoUpsert(dbname, collname, "dataset", records)
	if err != nil {
		t.Error(err)
	}
	spec = bson.M{"test": 1}
	records = MongoGet(dbname, collname, spec, idx, limit)
	if len(records) != 1 {
		t.Errorf("unable to find records using spec '%s', records %+v", spec, records)
	}
}
