package main

// handlers module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"fmt"
	"html/template"
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
	var user string
	var err error
	if Config.TestMode {
		user = "testUser"
	} else {
		user, err = username(r)
		if err != nil {
			LoginHandler(w, r)
			return
		}
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
	tmplData := make(map[string]interface{})
	tmplData["User"] = user
	tmplData["Date"] = time.Now().Unix()
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

// helper function to process input form
func processForm(r *http.Request) Record {
	rec := make(Record)
	rec["date"] = time.Now().Unix()
	// r.PostForm provides url.Values which is map[string][]string type
	// we convert it to Record
	arr := []string{"constituentelements", "phases", "experimenttype", "supplementarymeasurements", "xraytechnique", "loadframe", "detectors"}
	for k, items := range r.PostForm {
		k = strings.ToLower(k)
		if k == "proposal" || k == "btr" || k == "date" || k == "did" {
			if len(items) > 0 {
				v, e := strconv.ParseInt(items[0], 10, 64)
				if e == nil {
					rec[k] = v
				}
			}
		} else if k == "energy" {
			if len(items) > 0 {
				v, e := strconv.ParseFloat(items[0], 64)
				if e == nil {
					rec[k] = v
				}
			}
		} else if InList(k, arr) {
			if len(items) > 0 {
				arr := strings.Split(items[0], ",")
				rec[k] = arr
			}
		} else {
			if strings.Contains(k, "-") {
				kkk := strings.Split(k, "-")
				k := kkk[0]
				if vals, ok := rec[k]; ok {
					values := vals.([]string)
					for _, v := range items {
						values = append(values, v)
					}
					rec[k] = values
				} else {
					rec[k] = items
				}
			} else {
				for _, v := range items {
					rec[k] = v
					break
				}
			}
		}
	}
	logs.WithFields(logs.Fields{"record": rec}).Info("process form")
	return rec
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
		rec := processForm(r)
		err := insertData(rec)
		if err == nil {
			msg = fmt.Sprintf("Your meta-data is inserted successfully")
			class = "alert is-success"
		} else {
			msg = fmt.Sprintf("Web processing error: %v", err)
			class = "alert is-error"
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
		var data = Record{}
		data["user"] = user
		err = json.Unmarshal([]byte(config), &data)
		if err != nil {
			msg := "unable to unmarshal configuration data"
			handleError(w, r, msg, err)
			return
		}
		err = insertData(data)
		if err != nil {
			msg := "unable to insert data"
			handleError(w, r, msg, err)
			return
		}
		msg := fmt.Sprintf("Successfully inserted:\n%v", data.ToString())
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
		var data = Record{}
		data["user"] = user
		fmt.Println("body", string(body))
		err := json.Unmarshal(body, &data)
		if err != nil {
			msg = fmt.Sprintf("error: %v, unable to parse request data", err)
			class = "alert is-error"
		} else {
			err := insertData(data)
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
	var rec Record
	err := json.Unmarshal([]byte(record), &rec)
	if err != nil {
		msg := "unable to unmarshal passed record"
		handleError(w, r, msg, err)
		return
	}
	// we will prepare input entries for the template
	// where each entry represented in form of template.HTML
	// to avoid escaping of HTML characters
	var inputs []template.HTML
	var attrs []string
	for _, a := range Config.AdjustableAttrs {
		attrs = append(attrs, strings.ToLower(a))
	}
	for _, k := range MapKeys(rec) {
		var v string
		switch vvv := rec[k].(type) {
		case []string:
			v = strings.Join(vvv, ",")
		case []interface{}:
			var out []string
			for _, val := range vvv {
				out = append(out, fmt.Sprintf("%v", val))
			}
			v = strings.Join(out, ",")
		case int64, int:
			v = fmt.Sprintf("%d", vvv)
		case float64:
			d := int64(vvv)
			if float64(d) == vvv {
				v = fmt.Sprintf("%d", d)
			} else {
				v = fmt.Sprintf("%v", vvv)
			}
		default:
			v = fmt.Sprintf("%v", vvv)
		}
		in := fmt.Sprintf("<label>%s</label>", k)
		in = fmt.Sprintf("%s<input type=\"text\" name=\"%s\" value=\"%s\"", in, k, v)
		if InList(k, attrs) {
			in = fmt.Sprintf("%s class=\"is-90 is-success\"", in)
		} else {
			in = fmt.Sprintf("%s class=\"is-90\" readonly", in)
		}
		in = fmt.Sprintf("%s>", in)
		inputs = append(inputs, template.HTML(in))
	}
	tmplData["Inputs"] = inputs
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
	if err := r.ParseForm(); err == nil {
		rec = processForm(r)
		rid := r.FormValue("_id")
		// delete record id before the update
		delete(rec, "_id")
		msg = fmt.Sprintf("record %v is successfully updated", rid)
		fmt.Println("MongoUpsert", rec)
		records := []Record{rec}
		err = MongoUpsert(Config.DBName, Config.DBColl, records)
		if err != nil {
			msg = fmt.Sprintf("record %v update is failed, reason: %v", rid, err)
			cls = "is-error"
		} else {
			cls = "is-success"
		}
	} else {
		msg = fmt.Sprintf("record update failed, reason: %v", err)
		cls = "is-error"
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Message"] = strings.ToTitle(msg)
	tmplData["Class"] = fmt.Sprintf("alert %s is-large is-text-center", cls)
	page := templates.Confirm(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}
