package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

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

// helper function to create http test response recorder
// for given HTTP Method, url, reader and web handler
func respRecorder(method, url string, reader io.Reader, hdlr func(http.ResponseWriter, *http.Request)) (*httptest.ResponseRecorder, error) {
	// setup HTTP request
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	if method == "POST" {
		if strings.Contains(url, "search") || strings.Contains(url, "api") {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(hdlr)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		data, e := io.ReadAll(rr.Body)
		if e != nil {
			log.Println("unable to read reasponse body, error:", e)
		}
		log.Printf("handler returned status code: %v message: %s",
			status, string(data))
		msg := fmt.Sprintf("HTTP status %v", status)
		return nil, errors.New(msg)
	}
	return rr, nil
}

// TestHTTPGetApi provides test of GET method for our service
func TestHTTPGetApi(t *testing.T) {
	configFile := "server_test.json"
	ParseConfig(configFile)

	// setup HTTP request
	rr, err := respRecorder("GET", "/faq", nil, FAQHandler)
	if err != nil {
		t.Error(err)
	}

	data := rr.Body.Bytes()
	if !strings.Contains(string(data), "Frequently") {
		t.Error("FAQHandler content does not match")
	}
}

// TestHTTPPostGet provides test of POST/GET methods for our service
func TestHTTPPostGet(t *testing.T) {
	configFile := "server_test.json"
	ParseConfig(configFile)

	// use verbose log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// initialize FilesDB connection
	var err error
	FilesDB, err = InitFilesDB()
	defer FilesDB.Close()
	if err != nil {
		log.Printf("FilesDB error: %v\n", err)
	}
	initSchemaManager()

	// HTTP POST request
	schema := "test"
	inputRecord := `{"StringKey": "test","StrKeyMultipleValues": "foo,bla","ListKey": ["foo", "bla"],"FloatKey": 1.1,"BoolKey": true}`
	form := url.Values{}
	form.Add("record", string(inputRecord))
	form.Add("SchemaName", schema)
	reader := strings.NewReader(form.Encode())
	rr, err := respRecorder("POST", "/api", reader, APIHandler)
	if err != nil {
		t.Error(err)
	}

	data := rr.Body.Bytes()
	// unmarshal received records
	var record Record
	err = json.Unmarshal(data, &record)
	if err != nil {
		t.Errorf("unable to unmarshal received data '%s', error %v", string(data), err)
	}
	if v, ok := record["status"]; ok {
		status := v.(float64)
		if status != 200 {
			t.Errorf("wrong status code in record %+v", record)
		}
	} else {
		t.Error("no status code in record")
	}

	// HTTP GET request
	query := "user:test"
	form = url.Values{}
	form.Add("query", query)
	form.Add("client", "cli")
	reader = strings.NewReader(form.Encode())
	rr, err = respRecorder("GET", "/search", reader, SearchHandler)
	if err != nil {
		t.Error(err)
	}
	// unmarshal received records
	var rec Record
	err = json.Unmarshal(data, &rec)
	if err != nil {
		t.Errorf("unable to unmarshal received data '%s', error %v", string(data), err)
	}
	log.Printf("search reply record %+v", rec)
}

// TestHTTPBadRecord provides test of POST/GET methods for our service
func TestHTTPBadRecord(t *testing.T) {
	configFile := "server_test.json"
	ParseConfig(configFile)

	// use verbose log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// initialize FilesDB connection
	var err error
	FilesDB, err = InitFilesDB()
	defer FilesDB.Close()
	if err != nil {
		log.Printf("FilesDB error: %v\n", err)
	}
	initSchemaManager()

	// HTTP POST request
	schema := "test"
	inputRecord := `{"StringKey": true}`
	form := url.Values{}
	form.Add("record", string(inputRecord))
	form.Add("SchemaName", schema)
	reader := strings.NewReader(form.Encode())
	_, err = respRecorder("POST", "/api", reader, APIHandler)
	if err == nil {
		t.Error("No error thrown for injection of bad record")
	} else {
		log.Println("bad record injection returns", err)
	}
}
