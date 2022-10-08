package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/jcmturner/gokrb5.v7/client"
	"gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/credentials"
)

// helper functions for handlers module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

// helper function to extract username from auth-session cookie
func username(r *http.Request) (string, error) {
	cookie, err := r.Cookie("auth-session")
	if err != nil {
		return "", err
	}

	//     byteArray := decrypt([]byte(cookie.Value), Config.StoreSecret)
	//     n := bytes.IndexByte(byteArray, 0)
	//     s := string(byteArray[:n])

	s := cookie.Value

	arr := strings.Split(s, "-")
	if len(arr) != 2 {
		return "", errors.New("Unable to decript auth-session")
	}
	user := arr[0]
	return user, nil
}

// https://github.com/jcmturner/gokrb5/issues/7
func kuserFromCache(cacheFile string) (*credentials.Credentials, error) {
	cfg, err := config.Load(Config.Krb5Conf)
	ccache, err := credentials.LoadCCache(cacheFile)
	client, err := client.NewClientFromCCache(ccache, cfg)
	err = client.Login()
	if err != nil {
		return nil, err
	}
	return client.Credentials, nil

}

// helper function to perform kerberos authentication
func kuser(user, password string) (*credentials.Credentials, error) {
	cfg, err := config.Load(Config.Krb5Conf)
	if err != nil {
		log.Printf("reading krb5.conf failes, error %v\n", err)
		return nil, err
	}
	client := client.NewClientWithPassword(user, Config.Realm, password, cfg, client.DisablePAFXFAST(true))
	err = client.Login()
	if err != nil {
		log.Printf("client login fails, error %v\n", err)
		return nil, err
	}
	return client.Credentials, nil
}

// authentication function
func auth(r *http.Request) error {
	_, err := username(r)
	return err
}

// helper function to handle http server errors
func handleError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	log.Printf("Error %v\n", err)
	var templates Templates
	tmplData := make(map[string]interface{})
	tmplData["Message"] = strings.ToTitle(msg)
	tmplData["Class"] = "alert is-error is-large is-text-center"
	page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// helper function to check user credentials for POST requests
func getUserCredentials(r *http.Request) (*credentials.Credentials, error) {
	var msg string
	// user didn't use web interface, we switch to POST form
	name := r.FormValue("name")
	ticket := r.FormValue("ticket")
	tmpFile, err := ioutil.TempFile("/tmp", name)
	if err != nil {
		msg = fmt.Sprintf("Unable to create tempfile: %v", err)
		return nil, errors.New(msg)
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write([]byte(ticket))
	if err != nil {
		msg = "unable to write kerberos ticket"
		return nil, errors.New(msg)
	}
	err = tmpFile.Close()
	creds, err := kuserFromCache(tmpFile.Name())
	if err != nil {
		msg = "wrong user credentials"
		return nil, errors.New(msg)
	}
	if creds == nil {
		msg = "unable to obtain user credentials"
		return nil, errors.New(msg)
	}
	return creds, nil
}

// helper function to validate input data record against schema
func validateData(rec Record) error {
	var errs []string
	for _, smgr := range _smgr.Map {
		schema := smgr.Schema
		err := schema.Validate(rec)
		if err != nil {
			e := fmt.Sprintf("%s, error=%v", schema.String(), err)
			errs = append(errs, e)
		} else {
			log.Printf("Record %+v pass schema %s", rec, schema.String())
			return nil
		}
	}
	if len(errs) == len(_smgr.Map) {
		// we checked all schemas and all of them returned the error
		msg := fmt.Sprintf("Unable to find schema for this records\n%s", strings.Join(errs, "\n"))
		return errors.New(msg)
	}
	return nil
}

// helper function to preprocess given record
func preprocess(rec Record) Record {
	r := make(Record)
	for k, v := range rec {
		switch val := v.(type) {
		case string:
			r[strings.ToLower(k)] = strings.ToLower(val)
		case []string:
			var vals []string
			for _, vvv := range val {
				vals = append(vals, strings.ToLower(vvv))
			}
			r[strings.ToLower(k)] = vals
		case []interface{}:
			var vals []string
			for _, vvv := range val {
				s := fmt.Sprintf("%v", vvv)
				vals = append(vals, strings.ToLower(s))
			}
			r[strings.ToLower(k)] = vals
		default:
			r[strings.ToLower(k)] = val
		}
	}
	return r
}

// helper function to insert data into backend DB
func insertData(rec Record) error {
	// check if data satisfies to one of the schema
	if err := validateData(rec); err != nil {
		return err
	}
	if _, ok := rec["Date"]; !ok {
		rec["Date"] = time.Now().Unix()
	}
	// main attributes to work with
	// TODO: I need to decide if CHESS need experiment, processing, tier
	// and then a dataset
	// if not, then I will need to modify InsertFiles API to not use them
	var path, experiment, processing string
	if v, ok := rec["RawDataDirectory"]; ok {
		path = v.(string)
	} else {
		path = "test"
	}
	if v, ok := rec["AbbreviatedName"]; ok {
		experiment = v.(string)
	} else {
		experiment = "CHESS"
	}
	if v, ok := rec["Processing"]; ok {
		processing = v.(string)
	} else {
		processing = "processing"
	}
	tier := "raw"
	dataset := fmt.Sprintf("/%s/%s/%s", experiment, processing, tier)
	rec["dataset"] = dataset
	rec = preprocess(rec)
	// check if given path exist on file system
	_, err := os.Stat(path)
	if err == nil {
		log.Printf("input data, record %v, path %v\n", rec, path)
		rec["path"] = path
		// we generate unique id by using time stamp
		did := time.Now().UnixNano()
		go InsertFiles(did, experiment, processing, tier, path)
		rec["did"] = did
		records := []Record{rec}
		MongoUpsert(Config.DBName, Config.DBColl, records)
		return nil
	}
	msg := fmt.Sprintf("No files found associated with path=%s", path)
	return errors.New(msg)
}
