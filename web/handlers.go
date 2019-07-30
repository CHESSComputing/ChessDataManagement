package main

// handlers module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//
// OAuth tutorials:
// https://github.com/sohamkamani/go-oauth-example
// https://jacobmartins.com/2016/02/29/getting-started-with-oauth2-in-go/
// https://www.sohamkamani.com/blog/golang/2018-06-24-oauth-with-golang/
// https://auth0.com/docs/quickstart/webapp/golang/01-login
// https://developer.github.com/apps/building-oauth-apps/authorizing-oauth-apps

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	logs "github.com/sirupsen/logrus"
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

// OAuthAccessResponse provides sturcture to hold access token
type OAuthAccessResponse struct {
	AccessToken string `json:"access_token"`
}

// helper function to get username from the session
func username(r *http.Request) (string, error) {
	session, err := Store.Get(r, "auth-session")
	if err != nil {
		return "", err
	}
	if user, status := session.Values["user"]; status {
		logs.WithFields(logs.Fields{
			"User": user,
		}).Debug("authenticated")
		return user.(string), nil
	}
	return "", errors.New("User not found")
}

// authentication function
func auth(r *http.Request) error {
	_, err := username(r)
	return err
}

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
	err := auth(r)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not authenticate")
		LoginHandler(w, r)
		return
	}
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
	tmplData["ClientID"] = Config.ClientID
	tmplData["Redirect"] = Config.Redirect
	page := templates.LoginForm(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// OAuthHandler provides OAuth authentication to our app
func OAuthHandler(w http.ResponseWriter, r *http.Request) {
	httpClient := http.Client{}

	// First, we need to get the value of the `code` query param
	err := r.ParseForm()
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not parse the query")
		w.WriteHeader(http.StatusBadRequest)
	}
	code := r.FormValue("code")

	// Next, lets for the HTTP request to call the github oauth enpoint
	// to get our access token
	reqURL := fmt.Sprintf("https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s", Config.ClientID, Config.ClientSecret, code)
	req, err := http.NewRequest(http.MethodPost, reqURL, nil)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not create HTTP request")
		w.WriteHeader(http.StatusBadRequest)
	}
	// We set this header since we want the response
	// as JSON
	req.Header.Set("accept", "application/json")

	// Send out the HTTP request
	res, err := httpClient.Do(req)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not send HTTP request")
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer res.Body.Close()

	// Parse the request body into the `OAuthAccessResponse` struct
	var t OAuthAccessResponse
	if err := json.NewDecoder(res.Body).Decode(&t); err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not parse JSON response")
		w.WriteHeader(http.StatusBadRequest)
	}

	// get user info from github
	reqURL = fmt.Sprintf("https://api.github.com/user")
	req, err = http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not create HTTP request")
		w.WriteHeader(http.StatusBadRequest)
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %s", t.AccessToken))
	res, err = httpClient.Do(req)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not send HTTP request")
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer res.Body.Close()
	var userInfo map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("could not parse JSON response")
		w.WriteHeader(http.StatusBadRequest)
	}

	// add user credentials to store
	session, err := Store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session.Values["user"] = userInfo["login"].(string)
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logs.WithFields(logs.Fields{
		"UserInfo": userInfo,
		"Token":    t.AccessToken,
	}).Debug("oauth")

	// Finally, send a response to redirect the user to the "welcome" page
	// with the access token
	//w.Header().Set("Location", "/data?access_token="+t.AccessToken)
	w.Header().Set("Location", "/data")
	w.WriteHeader(http.StatusFound)
}

// SearchHandler handlers Search requests
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	page := templates.SearchForm(Config.Templates, tmplData)

	var records []Record
	var nrec int // we'll use it for pagination later
	if err := r.ParseForm(); err == nil {
		// r.PostForm provides url.Values which is map[string][]string type
		// we convert it to Record
		query := r.PostForm["query"]
		spec := ParseQuery(query)
		if spec != nil {
			nrec = MongoCount(Config.DBName, Config.DBColl, spec)
			records = MongoGet(Config.DBName, Config.DBColl, spec, 0, -1)
		}
		logs.WithFields(logs.Fields{
			"Spec":    spec,
			"Records": records,
		}).Debug("results")
	}
	// TODO: implement pagination
	page = fmt.Sprintf("%s</br></br>Found %d results</br>", page, nrec)
	for _, rec := range records {
		oid := rec["_id"].(bson.ObjectId)
		tmplData["Id"] = oid.Hex()
		tmplData["Record"] = rec.ToString()
		prec := templates.Record(Config.Templates, tmplData)
		page = fmt.Sprintf("%s</br>%s", page, prec)
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
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("VerboseHandler unable to marshal", err)
		w.WriteHeader(http.StatusInternalServerError)
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
				class = "msg-error"
			} else {
				records := []Record{rec}
				MongoUpsert(Config.DBName, Config.DBColl, records)
				msg = fmt.Sprintf("SUCCESS:\n\nMETA-DATA:\n%v\n\nDATASET: %s contains %d files", rec.ToString(), dataset, len(files))
				class = "msg-success"
			}
		} else {
			msg = fmt.Sprintf("WARNING:\nUnable to find any files in given path '%s'", path)
			class = "msg-warning"
		}
	} else {
		msg = fmt.Sprintf("ERROR:\nWeb processing error: %v", err)
		class = "msg-error"
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Message"] = msg
	tmplData["Class"] = class
	page := templates.Confirm(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}
