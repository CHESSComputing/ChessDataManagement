package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
		req.Header.Set("Content-Type", "application/json")
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

// TestHTTPGet provides test of GET method for our service
func TestHTTPGet(t *testing.T) {
	configFile := "server.json"
	ParseConfig(configFile)

	// setup HTTP request
	rr, err := respRecorder("GET", "/faq", nil, FAQHandler)
	if err != nil {
		t.Error(err)
	}

	data := rr.Body.Bytes()
	fmt.Println("FAQHandler returns", string(data))
	if !strings.Contains(string(data), "Frequently") {
		t.Error("FAQHandler content does not match")
	}
	// unmarshal received records
	//     var records []Record
	//     err = json.Unmarshal(data, &records)
	//     if err != nil {
	//         t.Errorf("unable to unmarshal received data '%s', error %v", string(data), err)
	//     }
}
