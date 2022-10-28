package main

// handlers module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
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
	log.Printf("User %v path %v\n", user, r.URL.Path)
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
	case "files":
		FilesHandler(w, r)
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
	var templates Templates
	tmplData := make(map[string]interface{})
	page := templates.Tmpl(Config.Templates, "login.tmpl", tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// KAuthHandler provides KAuth authentication to our app
func KAuthHandler(w http.ResponseWriter, r *http.Request) {
	// First, we need to get the value of the `code` query param
	err := r.ParseForm()
	if err != nil {
		log.Printf("could not parse http form, error %v\n", err)
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
	var templates Templates
	tmplData := make(map[string]interface{})
	tmplData["Query"] = query
	page := templates.Tmpl(Config.Templates, "searchform.tmpl", tmplData)

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
			tmplData["Did"] = rec["did"]
			tmplData["RecordString"] = rec.ToString()
			tmplData["Record"] = rec.ToJSON()
			prec := templates.Tmpl(Config.Templates, "record.tmpl", tmplData)
			page = fmt.Sprintf("%s<br>%s", page, prec)
		}
		if nrec > 5 {
			page = fmt.Sprintf("%s<br><br>%s", page, pager)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// helper function to generate input form
func genForm(fname string) (string, error) {
	var out []string
	val := fmt.Sprintf("<h3>Web form submission</h3><br/>")
	out = append(out, val)
	beamline := fileName(fname)
	val = fmt.Sprintf("<input name=\"beamline\" type=\"hidden\" value=\"\"/>%s", beamline)
	schema, err := _smgr.Load(fname)
	if err != nil {
		log.Println("unable to load", fname, "error", err)
		return strings.Join(out, ""), err
	}
	optKeys, err := schema.OptionalKeys()
	if err != nil {
		log.Println("unable to get optional keys, error", err)
		return strings.Join(out, ""), err
	}
	allKeys, err := schema.Keys()
	if err != nil {
		log.Println("unable to get keys, error", err)
		return strings.Join(out, ""), err
	}
	sectionKeys, err := schema.SectionKeys()
	if err != nil {
		log.Println("unable to get section keys, error", err)
		return strings.Join(out, ""), err
	}

	// loop over all defined sections
	var rec string
	sections, err := schema.Sections()
	if err != nil {
		log.Println("unable to get sections, error", err)
		return strings.Join(out, ""), err
	}
	for _, s := range sections {
		if skeys, ok := sectionKeys[s]; ok {
			showSection := false
			if len(skeys) != 0 {
				showSection = true
			}
			if showSection {
				out = append(out, fmt.Sprintf("<fieldset id=\"%s\">", s))
				out = append(out, fmt.Sprintf("<legend>%s</legend>", s))
			}
			for _, k := range skeys {
				if InList(k, optKeys) {
					rec = formEntry(schema.Map, k, s, "")
				} else {
					rec = formEntry(schema.Map, k, s, "required")
				}
				out = append(out, rec)
			}
			if showSection {
				out = append(out, "</fieldset>")
			}
		}
	}
	// loop over the rest of section keys which did not show up in sections
	for s, skeys := range sectionKeys {
		if InList(s, sections) {
			continue
		}
		showSection := false
		if len(skeys) != 0 {
			showSection = true
		}
		if showSection {
			out = append(out, fmt.Sprintf("<fieldset id=\"%s\">", s))
			out = append(out, fmt.Sprintf("<legend>%s</legend>", s))
		}
		for _, k := range skeys {
			if InList(k, optKeys) {
				rec = formEntry(schema.Map, k, s, "required")
			} else {
				rec = formEntry(schema.Map, k, s, "")
			}
			out = append(out, rec)
		}
		if showSection {
			out = append(out, "</fieldset>")
		}
	}
	// loop over all keys which do not have sections
	var nOut []string
	for _, k := range allKeys {
		if r, ok := schema.Map[k]; ok {
			if r.Section == "" {
				if InList(k, optKeys) {
					rec = formEntry(schema.Map, k, "", "")
				} else {
					rec = formEntry(schema.Map, k, "", "required")
				}
				nOut = append(nOut, rec)
			}
		}
	}
	if len(nOut) > 0 {
		out = append(out, fmt.Sprintf("<fieldset id=\"attributes\">"))
		out = append(out, "<legend>Attriburtes</legend>")
		for _, rec := range nOut {
			out = append(out, rec)
		}
		out = append(out, "</fieldset>")
	}
	form := strings.Join(out, "\n")
	tmplData := make(map[string]interface{})
	tmplData["Beamline"] = beamline
	tmplData["Form"] = template.HTML(form)
	var templates Templates
	return templates.Tmpl(Config.Templates, "form_beamline.tmpl", tmplData), nil
}

// helper function to create form entry
func formEntry(smap map[string]SchemaRecord, k, s, required string) string {
	tmplData := make(map[string]interface{})
	tmplData["Key"] = k
	tmplData["Value"] = ""
	tmplData["Placeholder"] = ""
	tmplData["Description"] = ""
	tmplData["Required"] = required
	if required != "" {
		tmplData["Class"] = "is-req"
	}
	tmplData["Type"] = "text"
	tmplData["Multiple"] = ""
	if r, ok := smap[k]; ok {
		if r.Section == s {
			if r.Type == "list_str" || r.Type == "list_int" || r.Type == "list_float" || r.Type == "list" {
				tmplData["List"] = true
				switch values := r.Value.(type) {
				case []any:
					var vals []string
					for _, v := range values {
						vals = append(vals, fmt.Sprintf("%v", v))
					}
					tmplData["Value"] = vals
					//                     tmplData["Value"] = values
				default:
					tmplData["Value"] = []string{}
				}
			} else if r.Type == "bool" || r.Type == "boolean" {
				tmplData["List"] = true
				if r.Value == true {
					tmplData["Value"] = []string{"", "true", "false"}
				} else {
					tmplData["Value"] = []string{"", "false", "true"}
				}
			} else {
				if r.Value != nil {
					//                     tmplData["Value"] = fmt.Sprintf("%v", r.Value)
					switch values := r.Value.(type) {
					case []any:
						var vals []string
						for _, v := range values {
							vals = append(vals, fmt.Sprintf("%v", v))
						}
						tmplData["Value"] = vals
					default:
						tmplData["Value"] = fmt.Sprintf("%v", r.Value)
					}
				}
			}
			if r.Multiple {
				tmplData["Multiple"] = "multiple"
			}
			desc := fmt.Sprintf("%s", r.Description)
			if desc == "" {
				desc = "Not Available"
			}
			tmplData["Description"] = desc
			tmplData["Placeholder"] = r.Placeholder
		}
	}
	var templates Templates
	return templates.Tmpl(Config.Templates, "form_entry.tmpl", tmplData)
}

// DataHandler handlers Data requests
func DataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	user, _ := username(r)
	var templates Templates
	tmplData := make(map[string]interface{})
	tmplData["User"] = user
	tmplData["Date"] = time.Now().Unix()
	tmplData["Beamlines"] = _beamlines
	var forms []string
	for idx, fname := range Config.SchemaFiles {
		cls := "hide"
		if idx == 0 {
			cls = ""
		}
		form, err := genForm(fname)
		if err != nil {
			log.Println("ERROR", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		beamlineForm := fmt.Sprintf("<div id=\"%s\" class=\"%s\">%s</div>", fileName(fname), cls, form)
		forms = append(forms, beamlineForm)
	}
	tmplData["Form"] = template.HTML(strings.Join(forms, "\n"))
	page := templates.Tmpl(Config.Templates, "keys.tmpl", tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// FAQHandler handlers FAQ requests
func FAQHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var templates Templates
	tmplData := make(map[string]interface{})
	page := templates.Tmpl(Config.Templates, "faq.tmpl", tmplData)
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
	var templates Templates
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
	page := templates.Tmpl(Config.Templates, "status.tmpl", tmplData)
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
	log.Printf("update server settings, level %v\n", s.Level)
	w.WriteHeader(http.StatusOK)
	return
}

// helper function to parser form values
func parseValue(schema *Schema, key string, items []string) (any, error) {
	r, ok := schema.Map[key]
	if !ok {
		if Config.TestMode && InList(key, _skipKeys) {
			return "", nil
		}
		msg := fmt.Sprintf("No key %s found in schema map", key)
		return false, errors.New(msg)
	} else if r.Type == "list_str" {
		return items, nil
	} else if strings.HasPrefix(r.Type, "list_int") {
		return items, nil
	} else if strings.HasPrefix(r.Type, "list_float") {
		return items, nil
	} else if r.Type == "string" {
		return items[0], nil
	} else if r.Type == "bool" {
		v, err := strconv.ParseBool(items[0])
		if err == nil {
			return v, nil
		}
		msg := fmt.Sprintf("Unable to parse boolean value for key=%s, please come back to web form and choose either true or false", key)
		return false, errors.New(msg)
	} else if strings.HasPrefix(r.Type, "int") {
		v, err := strconv.ParseInt(items[0], 10, 64)
		if err == nil {
			return v, nil
		}
		return 0, err
	} else if strings.HasPrefix(r.Type, "float") {
		v, err := strconv.ParseFloat(items[0], 64)
		if err == nil {
			return v, nil
		}
		return 0, err
	}
	msg := fmt.Sprintf("Unable to parse form value for key %s", key)
	return 0, errors.New(msg)
}

// helper function to process input form
func processForm(r *http.Request) (Record, error) {
	rec := make(Record)
	rec["Date"] = time.Now().Unix()
	// read schemaName from form itself
	var sname string
	for k, items := range r.PostForm {
		if k == "SchemaName" {
			sname = items[0]
			break
		}
	}
	var fname string
	for _, f := range Config.SchemaFiles {
		if strings.Contains(f, sname) {
			fname = f
			break
		}
	}
	schema, err := _smgr.Load(fname)
	if err != nil {
		log.Println("ERROR", err)
		return rec, err
	}
	// r.PostForm provides url.Values which is map[string][]string type
	// we convert it to Record
	for k, items := range r.PostForm {
		if Config.Verbose > 0 {
			log.Println("### PostForm", k, items)
		}
		if k == "SchemaName" {
			continue
		}
		val, err := parseValue(schema, k, items)
		if err != nil {
			// check if given key is mandatory or optional
			srec, ok := schema.Map[k]
			if ok {
				if srec.Optional {
					log.Println("WARNING: unable to parse optional key", k)
				} else {
					log.Println("ERROR: unable to parse mandatory key", k, "error", err)
					return rec, err
				}
			} else {
				log.Println("ERROR: no key", k, "found in schema map, error", err)
				return rec, err
			}
		}
		rec[k] = val
	}
	log.Printf("process form, record %v\n", rec)
	return rec, nil
}

// ProcessHandler handlers Process requests
func ProcessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var msg string
	var class string
	var templates Templates
	tmplData := make(map[string]interface{})
	if err := r.ParseForm(); err == nil {
		rec, err := processForm(r)
		if err != nil {
			msg = fmt.Sprintf("Web processing error: %v", err)
			class = "alert is-error"
			tmplData["Message"] = msg
			tmplData["Class"] = class
			page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(_top + page + _bottom))
			return
		}
		err = insertData(rec)
		if err == nil {
			msg = fmt.Sprintf("Your meta-data is inserted successfully")
			log.Println("INFO", msg)
			class = "alert is-success"
		} else {
			msg = fmt.Sprintf("Web processing error: %v", err)
			class = "alert is-error"
			log.Println("WARNING", msg)
			tmplData["Message"] = msg
			tmplData["Class"] = class
			page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
			// redirect users to update their record
			inputs := htmlInputs(rec)
			tmplData["Inputs"] = inputs
			tmplData["Id"] = ""
			page += templates.Tmpl(Config.Templates, "update.tmpl", tmplData)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(_top + page + _bottom))
			return
		}
	}
	tmplData["Message"] = msg
	tmplData["Class"] = class
	page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// APIHandler handlers Api requests
func APIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var user string
	var err error
	if Config.TestMode {
		user = "test"
	} else {
		user, err = username(r)
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
			data["User"] = user
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
		data["User"] = user
		log.Println("body", string(body))
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
				msg = fmt.Sprintf("Api web processing error: %v", err)
				class = "alert is-error"
			}
		}
	}
	var templates Templates
	tmplData := make(map[string]interface{})
	tmplData["Message"] = msg
	tmplData["Class"] = class
	page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

func htmlInputs(rec Record) []template.HTML {
	var inputs []template.HTML
	// use attrs to adjust html form
	// it was user for adjustable attributes
	var attrs []string
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
		if strings.Contains(v, "ERROR") {
			in = fmt.Sprintf("<label class=\"is-req\">%s</label>", k)
			v = strings.Trim(strings.Replace(v, "ERROR", "", -1), " ")
			in = fmt.Sprintf("%s<input required class=\"alert is-error is-90\" type=\"text\" name=\"%s\" value=\"%s\"", in, k, v)
		} else {
			in = fmt.Sprintf("%s<input type=\"text\" name=\"%s\" value=\"%s\"", in, k, v)
			if InList(k, attrs) {
				in = fmt.Sprintf("%s class=\"is-90 is-success\"", in)
			} else {
				in = fmt.Sprintf("%s class=\"is-90\" readonly", in)
			}
		}
		in = fmt.Sprintf("%s>", in)
		inputs = append(inputs, template.HTML(in))
	}
	return inputs
}

// UpdateHandler handlers Process requests
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var templates Templates
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
	inputs := htmlInputs(rec)
	tmplData["Inputs"] = inputs
	tmplData["Id"] = r.FormValue("_id")
	page := templates.Tmpl(Config.Templates, "update.tmpl", tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// UpdateRecordHandler handlers Process requests
func UpdateRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var templates Templates
	tmplData := make(map[string]interface{})
	var msg, cls string
	var rec Record
	if err := r.ParseForm(); err == nil {
		rec, err = processForm(r)
		if err != nil {
			msg := fmt.Sprintf("Web processing error: %v", err)
			class := "alert is-error"
			tmplData["Message"] = msg
			tmplData["Class"] = class
			page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(_top + page + _bottom))
			return
		}
		rid := r.FormValue("_id")
		// delete record id before the update
		delete(rec, "_id")
		if rid == "" {
			err := insertData(rec)
			if err == nil {
				msg = fmt.Sprintf("Your meta-data is inserted successfully")
				cls = "alert is-success"
			} else {
				msg = fmt.Sprintf("update web processing error: %v", err)
				cls = "alert is-error"
			}
		} else {
			msg = fmt.Sprintf("record %v is successfully updated", rid)
			log.Println("MongoUpsert", rec)
			records := []Record{rec}
			err = MongoUpsert(Config.DBName, Config.DBColl, records)
			if err != nil {
				msg = fmt.Sprintf("record %v update is failed, reason: %v", rid, err)
				cls = "is-error"
			} else {
				cls = "is-success"
			}
		}
	} else {
		msg = fmt.Sprintf("record update failed, reason: %v", err)
		cls = "is-error"
	}
	tmplData["Message"] = strings.ToTitle(msg)
	tmplData["Class"] = fmt.Sprintf("alert %s is-large is-text-center", cls)
	page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// FilesHandler handlers Files requests
func FilesHandler(w http.ResponseWriter, r *http.Request) {
	_, err := username(r)
	if err != nil {
		_, err := getUserCredentials(r)
		if err != nil {
			msg := "unable to get user credentials"
			handleError(w, r, msg, err)
			return
		}
		did, err := strconv.ParseInt(r.FormValue("did"), 10, 64)
		if err != nil {
			msg := fmt.Sprintf("Unable to parse did\nError: %v", err)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(msg))
			return
		}
		files, err := getFiles(did)
		if err != nil {
			msg := fmt.Sprintf("Unable to get files\nError: %v", err)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(msg))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strings.Join(files, "\n")))
		return
	}
	var templates Templates
	tmplData := make(map[string]interface{})
	did, err := strconv.ParseInt(r.FormValue("did"), 10, 64)
	if err != nil {
		tmplData["Message"] = fmt.Sprintf("Unable to parse did\nError: %v", err)
		tmplData["Class"] = "alert is-error is-large is-text-center"
		page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(_top + page + _bottom))
		return
	}
	files, err := getFiles(did)
	if err != nil {
		tmplData["Message"] = fmt.Sprintf("Unable to query FilesDB\nError: %v", err)
		tmplData["Class"] = "alert is-error is-large is-text-center"
		page := templates.Tmpl(Config.Templates, "confirm.tmpl", tmplData)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(_top + page + _bottom))
		return
	}
	tmplData["Id"] = r.FormValue("_id")
	tmplData["Did"] = did
	tmplData["Files"] = strings.Join(files, "\n")
	page := templates.Tmpl(Config.Templates, "files.tmpl", tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}
