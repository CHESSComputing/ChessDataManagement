package main

import (
	"encoding/json"
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

// AuthHandler authenticate incoming requests and route them to appropriate handler
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	/*
		// check if server started with hkey file (auth is required)
		if config.Config.Hkey != "" {
			status := _cmsAuth.CheckAuthnAuthz(r.Header)
			if !status {
				msg := "You are not allowed to access this resource"
				http.Error(w, msg, http.StatusForbidden)
				return
			}
		}
	*/
	// increment GET/POST counters
	if r.Method == "GET" {
		atomic.AddUint64(&TotalGetRequests, 1)
	}
	if r.Method == "POST" {
		atomic.AddUint64(&TotalPostRequests, 1)
	}

	// check if server started with hkey file (auth is required)
	//     status := auth(r)
	//     if !status {
	//         msg := "You are not allowed to access this resource"
	//         http.Error(w, msg, http.StatusForbidden)
	//         return
	//     }
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
	default:
		DataHandler(w, r)
	}
}

// GET Methods

// SearchHandler handlers Search requests
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	page := templates.SearchForm(Config.Templates, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_top + page + _bottom))
}

// DataHandler handlers Data requests
func DataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Keys"] = []string{"Experiment", "Annotations"}
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
	virt := Memory{Total: m.Total, Free: m.Free, Used: m.Used, UsedPercent: m.UsedPercent}
	swap := Memory{Total: s.Total, Free: s.Free, Used: s.Used, UsedPercent: s.UsedPercent}
	tmplData["Memory"] = Mem{Virtual: virt, Swap: swap}
	tmplData["Load"] = l
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
	tmplData["getRequests"] = TotalGetRequests
	tmplData["postRequests"] = TotalPostRequests
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
	}).Info("update server settings")
	w.WriteHeader(http.StatusOK)
	return
}
