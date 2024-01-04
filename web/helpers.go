package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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
	if Config.TestMode {
		return "test", nil
	}
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
		msg := "Unable to decript auth-session"
		log.Printf("ERROR: %s", msg)
		return "", errors.New(msg)
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
	log.Printf("ERROR: %v\n", err)
	var templates Templates
	tmplData := makeTmplData()
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
		log.Printf("ERROR: %s", msg)
		return nil, errors.New(msg)
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write([]byte(ticket))
	if err != nil {
		msg = "unable to write kerberos ticket"
		log.Printf("ERROR: %s", msg)
		return nil, errors.New(msg)
	}
	err = tmpFile.Close()
	creds, err := kuserFromCache(tmpFile.Name())
	if err != nil {
		msg = "wrong user credentials"
		log.Printf("ERROR: %s", msg)
		return nil, errors.New(msg)
	}
	if creds == nil {
		msg = "unable to obtain user credentials"
		log.Printf("ERROR: %s", msg)
		return nil, errors.New(msg)
	}
	return creds, nil
}

// helper function to validate input data record against schema
func validateData(sname string, rec Record) error {
	if smgr, ok := _smgr.Map[sname]; ok {
		schema := smgr.Schema
		err := schema.Validate(rec)
		if err != nil {
			return err
		}
	} else {
		msg := fmt.Sprintf("No schema '%s' found for your record", sname)
		log.Printf("ERROR: %s, schema map %+v", msg, _smgr.Map)
		return errors.New(msg)
	}
	return nil
}

// helper function to preprocess given record
/*
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
*/

// helper function to insert data into backend DB
func insertData(sname string, rec Record) error {
	// load our schema
	if _, err := _smgr.Load(sname); err != nil {
		msg := fmt.Sprintf("unable to load %s error %v", sname, err)
		log.Println("ERROR: ", msg)
		return errors.New(msg)
	}

	// check if data satisfies to one of the schema
	if err := validateData(sname, rec); err != nil {
		return err
	}
	if _, ok := rec["Date"]; !ok {
		rec["Date"] = time.Now().Unix()
	}
	rec["SchemaFile"] = sname
	rec["Schema"] = schemaName(sname)
	// main attributes to work with
	var path, cycle, beamline, btr, sample string
	if v, ok := rec["DataLocationRaw"]; ok {
		path = v.(string)
	} else {
		path = filepath.Join("/tmp", os.Getenv("USER")) // for testing purposes
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("Directory %s does not exist, will use /tmp", path)
			path = "/tmp"
		}
	}
	// log record just in case we need to debug it
	log.Printf("cycle=%v beamline=%v btr=%v sample=%v", rec["Cycle"], rec["Beamline"], rec["BTR"], rec["SampleName"])
	if v, ok := rec["Cycle"]; ok {
		cycle = v.(string)
	} else {
		cycle = fmt.Sprintf("Cycle-%s", randomString())
	}
	if v, ok := rec["Beamline"]; ok {
		switch b := v.(type) {
		case string:
			beamline = b
		case []string:
			beamline = strings.Join(b, "-")
		case []any:
			var values []string
			for _, v := range b {
				values = append(values, fmt.Sprintf("%v", v))
			}
			beamline = strings.Join(values, "-")
		}
	} else {
		beamline = fmt.Sprintf("beamline-%s", randomString())
	}
	if v, ok := rec["BTR"]; ok {
		btr = v.(string)
	} else {
		btr = fmt.Sprintf("btr-%s", randomString())
	}
	if v, ok := rec["SampleName"]; ok {
		sample = v.(string)
	} else {
		sample = fmt.Sprintf("sample-%s", randomString())
	}
	// dataset is a /cycle/beamline/BTR/sample
	dataset := fmt.Sprintf("/%s/%s/%s/%s", cycle, beamline, btr, sample)
	rec["dataset"] = dataset
	//     rec = preprocess(rec)
	// check if given path exist on file system
	_, err := os.Stat(path)
	if err == nil {
		log.Printf("input data, record %v, path %v\n", rec, path)
		rec["path"] = path
		// generate unique id
		var did string
		if v, ok := rec["did"]; !ok {
			if uuid, err := uuid.NewRandom(); err == nil {
				did = hex.EncodeToString(uuid[:])
			} else {
				did = fmt.Sprintf("%v", time.Now().UnixMilli())
			}
		} else {
			did = v.(string)
		}
		err = InsertFiles(did, dataset, path)
		if err != nil {
			log.Printf("ERROR: unable to InsertFiles for did=%v dataset=%s path=%s, error=%v", did, dataset, path, err)
			return err
		}
		rec["did"] = did
		records := []Record{rec}
		err = MongoUpsert(Config.DBName, Config.DBColl, "dataset", records)
		if err != nil {
			log.Printf("ERROR: unable to MongoUpsert for did=%v dataset=%s path=%s, error=%v", did, dataset, path, err)
		}
		return err
	}
	msg := fmt.Sprintf("No files found associated with DataLocationRaw=%s", path)
	log.Printf("ERROR: %s", msg)
	return errors.New(msg)
}
