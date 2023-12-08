package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now()
	return fmt.Sprintf("git={{VERSION}} go=%s date=%s", goVersion, tstamp)
}

func main() {
	var fname string
	flag.StringVar(&fname, "fname", "", "file name to read or write based on the action")
	var action string
	flag.StringVar(&action, "action", "", "action: export or import")
	var uri string
	flag.StringVar(&uri, "uri", "", "uri: export or import")
	var dbname string
	flag.StringVar(&dbname, "dbname", "", "database name")
	var collname string
	flag.StringVar(&collname, "collname", "", "collection name")
	var version bool
	flag.BoolVar(&version, "version", false, "Show version")
	flag.Parse()
	if version {
		fmt.Println("mongotools version:", info())
		return
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	process(uri, dbname, collname, action, fname)
}

type Record map[string]any

func importData(client *mongo.Client, dbname, collname, fname string) {
	log.Printf("import data using %s.%s from %s", dbname, collname, fname)
	c := client.Database(dbname).Collection(collname)
	var records []Record
	// read data from fname
	file, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, &records)
	if err != nil {
		log.Fatal(err)
	}
	for _, rec := range records {
		if _, err := c.InsertOne(context.TODO(), &rec); err != nil {
			log.Printf("Fail to insert record %v, error %v\n", rec, err)
		}
	}
	log.Printf("Successfully imported %d records from %s\n", len(records), fname)
}

func exportData(client *mongo.Client, dbname, collname, fname string) {
	log.Printf("export data using %s.%s to %s\n", dbname, collname, fname)
	c := client.Database(dbname).Collection(collname)
	var records []Record
	ctx := context.TODO()
	spec := bson.M{}
	opts := options.Find()
	cur, err := c.Find(ctx, spec, opts)
	if err != nil {
		log.Fatal(err)
	}
	cur.All(ctx, &records)
	log.Printf("found %d records\n", len(records))
	data, err := json.Marshal(records)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(fname, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Successfully exported %d records to %s\n", len(records), fname)
}

func process(uri, dbname, collname, action, fname string) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if action == "import" {
		importData(client, dbname, collname, fname)
	} else if action == "export" {
		exportData(client, dbname, collname, fname)
	}
}
