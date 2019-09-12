package main

// handlers module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	logs "github.com/sirupsen/logrus"
	"gopkg.in/jcmturner/gokrb5.v7/client"
	"gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/credentials"
	"gopkg.in/mgo.v2/bson"
)

// TotalGetRequests counts total number of GET requests received by the server
var TotalGetRequests uint64

// TotalPostRequests counts total number of POST requests received by the server
var TotalPostRequests uint64

// ServerSettings controls server parameters
type ServerSettings struct {
	Level        int    `json:"level"`        // verbosity level
	LogFormatter string `json:"logFormatter"` // logrus formatter
}

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
		msg := "reading krb5.conf fails"
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error(msg)
		return nil, err
	}
	client := client.NewClientWithPassword(user, Config.Realm, password, cfg, client.DisablePAFXFAST(true))
	err = client.Login()
	if err != nil {
		msg := "client login fails"
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error(msg)
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
	logs.WithFields(logs.Fields{
		"Error": err,
	}).Error(msg)
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Message"] = strings.ToTitle(msg)
	tmplData["Class"] = "alert is-error is-large is-text-center"
	page := templates.Confirm(Config.Templates, tmplData)
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

//
// ### HTTP METHODS
//

// AuthHandler authenticate incoming requests and route them to appropriate handler
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	// increment GET/POST counters
	if r.Method == "GET" {
		atomic.AddUint64(&TotalGetRequests, 1)
	}
	if r.Method == "POST" {
		atomic.AddUint64(&TotalPostRequests, 1)
	}

	// check if server started with hkey file (auth is required)
	user, err := username(r)
	if err != nil {
		LoginHandler(w, r)
		return
	}
	logs.WithFields(logs.Fields{"User": user, "Path": r.URL.Path}).Info("")
	// define all methods which requires authentication
	arr := strings.Split(r.URL.Path, "/")
	path := arr[len(arr)-1]
	switch path {
	case "faq":
		FAQHandler(w, r)
	case "status":
		StatusHandler(w, r)
	case "server":
		SettingsHandler(w, r)
	case "search":
		SearchHandler(w, r)
	case "data":
		DataHandler(w, r)
	case "process":
		ProcessHandler(w, r)
	case "update":
		UpdateHandler(w, r)
	case "updateRecord":
		UpdateRecordHandler(w, r)
	default:
		DataHandler(w, r)
	}
}

// GET Methods

// LoginHandler handlers Login requests
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	page := templates.LoginForm(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// KAuthHandler provides KAuth authentication to our app
func KAuthHandler(w http.ResponseWriter, r *http.Request) {
	// First, we need to get the value of the `code` query param
	err := r.ParseForm()
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not parse http form")
		w.WriteHeader(http.StatusBadRequest)
	}
	name := r.FormValue("name")
	password := r.FormValue("password")
	var creds *credentials.Credentials
	if name != "" && password != "" {
		creds, err = kuser(name, password)
		if err != nil {
			msg := "wrong user credentials"
			handleError(w, r, msg, err)
			return
		}
	} else {
		msg := "user/password is empty"
		handleError(w, r, msg, err)
		return
	}
	if creds == nil {
		msg := "unable to obtain user credentials"
		handleError(w, r, msg, err)
		return
	}

	expiration := time.Now().Add(24 * time.Hour)
	msg := fmt.Sprintf("%s-%v", creds.UserName(), creds.Authenticated())
	//     byteArray := encrypt([]byte(msg), Config.StoreSecret)
	//     n := bytes.IndexByte(byteArray, 0)
	//     s := string(byteArray[:n])
	cookie := http.Cookie{Name: "auth-session", Value: msg, Expires: expiration}
	http.SetCookie(w, &cookie)
	w.Header().Set("Location", "/data")
	w.WriteHeader(http.StatusFound)
}

// SearchHandler handlers Search requests
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	_, err := username(r)
	if err != nil {
		_, err := getUserCredentials(r)
		if err != nil {
			msg := "unable to get user credentials"
			handleError(w, r, msg, err)
			return
		}
		query := r.FormValue("query")

		// process the query
		spec := ParseQuery(query)
		var records []Record
		if spec != nil {
			records = MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
		}
		data, err := json.Marshal(records)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("unable to marshal data, error=%v", err)))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}
	// get form parameters
	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		limit = 50
	}
	idx, err := strconv.Atoi(r.FormValue("idx"))
	if err != nil {
		idx = 0
	}
	query := r.FormValue("query")

	// create search template form
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Query"] = query
	page := templates.SearchForm(Config.Templates, tmplData)

	// process the query
	spec := ParseQuery(query)
	if spec != nil {
		nrec := MongoCount(Config.DBName, Config.DBColl, spec)
		records := MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
		var pager string
		if nrec > 0 {
			pager = pagination(query, nrec, idx, limit)
			page = fmt.Sprintf("%s<br><br>%s", page, pager)
		} else {
			page = fmt.Sprintf("%s<br><br>No results found</br>", page)
		}
		for _, rec := range records {
			oid := rec["_id"].(bson.ObjectId)
			rec["_id"] = oid
			tmplData["Id"] = oid.Hex()
			tmplData["RecordString"] = rec.ToString()
			tmplData["Record"] = rec.ToJson()
			prec := templates.Record(Config.Templates, tmplData)
			page = fmt.Sprintf("%s<br>%s", page, prec)
		}
		if nrec > 5 {
			page = fmt.Sprintf("%s<br><br>%s", page, pager)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// DataHandler handlers Data requests
func DataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user, _ := username(r)
	var templates ServerTemplates
	keysData := make(map[string]string)
	keysData["Experiment"] = "Name of the experiment"
	keysData["Processing"] = "processing version, e.g. tag-123, gcc-700"
	keysData["Tier"] = "data-tier, e.g. RAW"
	keysData["Run"] = "run number or annotation"
	keysData["Path"] = "input directory of experiment's files"
	tmplData := make(map[string]interface{})
	tmplData["Keys"] = keysData
	tmplData["User"] = user
	page := templates.Keys(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// FAQHandler handlers FAQ requests
func FAQHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	page := templates.FAQ(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// Memory structure keeps track of server memory
type Memory struct {
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

// Mem structure keeps track of virtual/swap memory of the server
type Mem struct {
	Virtual Memory
	Swap    Memory
}

// StatusHandler handlers Status requests
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// check HTTP header
	var accept, content string
	if _, ok := r.Header["Accept"]; ok {
		accept = r.Header["Accept"][0]
	}
	if _, ok := r.Header["Content-Type"]; ok {
		content = r.Header["Content-Type"][0]
	}

	// get cpu and mem profiles
	m, _ := mem.VirtualMemory()
	s, _ := mem.SwapMemory()
	l, _ := load.Avg()
	c, _ := cpu.Percent(time.Millisecond, true)
	process, perr := process.NewProcess(int32(os.Getpid()))

	// get unfinished queries
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["NGo"] = runtime.NumGoroutine()
	//     virt := Memory{Total: m.Total, Free: m.Free, Used: m.Used, UsedPercent: m.UsedPercent}
	//     swap := Memory{Total: s.Total, Free: s.Free, Used: s.Used, UsedPercent: s.UsedPercent}
	tmplData["Memory"] = m.UsedPercent
	tmplData["Swap"] = s.UsedPercent
	tmplData["Load1"] = l.Load1
	tmplData["Load5"] = l.Load5
	tmplData["Load15"] = l.Load15
	tmplData["CPU"] = c
	if perr == nil { // if we got process info
		conn, err := process.Connections()
		if err == nil {
			tmplData["Connections"] = conn
		}
		openFiles, err := process.OpenFiles()
		if err == nil {
			tmplData["OpenFiles"] = openFiles
		}
	}
	tmplData["Uptime"] = time.Since(Time0).Seconds()
	tmplData["GetRequests"] = TotalGetRequests
	tmplData["PostRequests"] = TotalPostRequests
	page := templates.Status(Config.Templates, tmplData)
	if strings.Contains(accept, "json") || strings.Contains(content, "json") {
		data, err := json.Marshal(tmplData)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("unable to marshal data, error=%v", err)))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// POST methods

// SettingsHandler handlers Settings requests
func SettingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var s = ServerSettings{}
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		msg := "unable to marshal server settings"
		handleError(w, r, msg, err)
		return
	}
	if s.LogFormatter == "json" {
		logs.SetFormatter(&logs.JSONFormatter{})
	} else if s.LogFormatter == "text" {
		logs.SetFormatter(&logs.TextFormatter{})
	} else {
		logs.SetFormatter(&logs.TextFormatter{})
	}
	logs.WithFields(logs.Fields{
		"Verbose level": s.Level,
		"Log formatter": s.LogFormatter,
	}).Debug("update server settings")
	w.WriteHeader(http.StatusOK)
	return
}

// helper function to insert data into backend DB
func insertData(rec Record) error {
	// main attributes to work with
	path := rec["path"].(string)
	experiment := rec["experiment"].(string)
	processing := rec["processing"].(string)
	tier := rec["tier"].(string)

	//         files := FindFiles(path)
	files := []string{path}
	dataset := fmt.Sprintf("/%s/%s/%s", experiment, processing, tier)
	rec["dataset"] = dataset
	if len(files) > 0 {
		logs.WithFields(logs.Fields{
			"Record": rec,
			"Files":  files,
		}).Debug("input data")
		rec["path"] = files[0]
		did, err := InsertFiles(experiment, processing, tier, files)
		if err != nil {
			return err
		}
		rec["did"] = did
		records := []Record{rec}
		MongoUpsert(Config.DBName, Config.DBColl, records)
		return nil
	}
	msg := fmt.Sprintf("No files found associated with path=%s, experiment=%s, processing=%s, tier=%s", path, experiment, processing, tier)
	return errors.New(msg)
}

// ProcessHandler handlers Process requests
func ProcessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var msg string
	var class string
	if err := r.ParseForm(); err == nil {
		rec := make(Record)
		// r.PostForm provides url.Values which is map[string][]string type
		// we convert it to Record
		for k, items := range r.PostForm {
			for _, v := range items {
				rec[strings.ToLower(k)] = v
				break
			}
		}
		err := insertData(rec)
		if err == nil {
			msg = fmt.Sprintf("Your meta-data is inserted successfully")
			class = "alert is-success"
		} else {
			msg = fmt.Sprintf("Web processing error: %v", err)
			class = "alert is-error"
		}

		/*
				path := rec["path"].(string)
				//         files := FindFiles(path)
				files := []string{path}
				delete(rec, "path")
				experiment := rec["experiment"].(string)
				processing := rec["processing"].(string)
				tier := rec["tier"].(string)
				dataset := fmt.Sprintf("/%s/%s/%s", experiment, processing, tier)
				rec["dataset"] = dataset
				if len(files) > 0 {
					logs.WithFields(logs.Fields{
						"Record": rec,
						"Files":  files,
					}).Debug("input data")
					rec["path"] = files[0]
					did, err := InsertFiles(experiment, processing, tier, files)
					rec["did"] = did
					if err != nil {
						msg = fmt.Sprintf("ERROR:\nWeb processing error: %v", err)
						class = "alert is-error"
					} else {
						records := []Record{rec}
						MongoUpsert(Config.DBName, Config.DBColl, records)
						msg = fmt.Sprintf("SUCCESS:\n\nMETA-DATA:\n%v\n\nDATASET: %s contains %d files", rec.ToString(), dataset, len(files))
						class = "alert is-success"
					}
				} else {
					msg = fmt.Sprintf("WARNING:\nUnable to find any files in given path '%s'", path)
					class = "alert is-warning"
				}
			} else {
				msg = fmt.Sprintf("ERROR:\nWeb processing error: %v", err)
				class = "alert is-error"
			}
		*/
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Message"] = msg
	tmplData["Class"] = class
	page := templates.Confirm(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// ApiHandler handlers Api requests
func ApiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user, err := username(r)
	if err != nil {
		creds, err := getUserCredentials(r)
		if err != nil {
			msg := "unable to get user credentials"
			handleError(w, r, msg, err)
			return
		}
		user = creds.UserName()
		config := r.FormValue("config")
		var data = ChessMetaData{}
		data.User = user
		err = json.Unmarshal([]byte(config), &data)
		if err != nil {
			msg := "unable to unmarshal configuration data"
			handleError(w, r, msg, err)
			return
		}
		err = insertData(data.ToRecord())
		if err != nil {
			msg := "unable to insert data"
			handleError(w, r, msg, err)
			return
		}
		msg := fmt.Sprintf("Successfully inserted:\n%v", data.String())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(msg))
		return
	}

	// process web form
	file, _, err := r.FormFile("file")
	if err != nil {
		msg := "unable to read file form"
		handleError(w, r, msg, err)
		return
	}
	defer file.Close()

	var msg, class string
	defer r.Body.Close()
	//     body, err := ioutil.ReadAll(r.Body)
	body, err := ioutil.ReadAll(file)
	if err != nil {
		msg = fmt.Sprintf("error: %v, unable to read request data", err)
		class = "alert is-error"
	} else {
		var data = ChessMetaData{}
		data.User = user
		fmt.Println("body", string(body))
		err := json.Unmarshal(body, &data)
		if err != nil {
			msg = fmt.Sprintf("error: %v, unable to parse request data", err)
			class = "alert is-error"
		} else {
			err := insertData(data.ToRecord())
			if err == nil {
				msg = fmt.Sprintf("meta-data is inserted successfully")
				class = "alert is-success"
			} else {
				msg = fmt.Sprintf("Web processing error: %v", err)
				class = "alert is-error"
			}
		}
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Message"] = msg
	tmplData["Class"] = class
	page := templates.Confirm(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// UpdateHandler handlers Process requests
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	record := r.FormValue("record")
	tmplData["Record"] = record
	tmplData["Id"] = r.FormValue("_id")
	page := templates.Update(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// UpdateRecordHandler handlers Process requests
func UpdateRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var msg, cls string
	var rec Record
	err := json.Unmarshal([]byte(r.FormValue("record")), &rec)
	if err != nil {
		msg = fmt.Sprintf("record update failed, reason: %v", err)
		cls = "is-error"
	} else {
		rid := r.FormValue("_id")
		// delete record id before the update
		delete(rec, "_id")
		msg = fmt.Sprintf("record %v is successfully updated", rid)
		records := []Record{rec}
		err = MongoUpsert(Config.DBName, Config.DBColl, records)
		if err != nil {
			msg = fmt.Sprintf("record %v update is failed, reason: %v", rid, err)
			cls = "is-error"
		} else {
			cls = "is-success"
		}
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Message"] = strings.ToTitle(msg)
	tmplData["Class"] = fmt.Sprintf("alert %s is-large is-text-center", cls)
	page := templates.Confirm(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}
