package main

import (
	_ "expvar" // to be used for monitoring, see https://github.com/divan/expvarmon
	"fmt"
	"net/http"
	_ "net/http/pprof" // profiler, see https://golang.org/pkg/net/http/pprof/
	"time"

	logs "github.com/sirupsen/logrus"
)

// global variables
var _top, _bottom, _search string

// Time0 represents initial time when we started the server
var Time0 time.Time

// Server code
func Server(configFile string) {
	Time0 = time.Now()
	err := ParseConfig(configFile)
	if Config.LogFormatter == "json" {
		logs.SetFormatter(&logs.JSONFormatter{})
	} else if Config.LogFormatter == "text" {
		logs.SetFormatter(&logs.TextFormatter{})
	} else {
		logs.SetFormatter(&logs.JSONFormatter{})
	}
	if err != nil {
		logs.WithFields(logs.Fields{"Time": time.Now(), "Config": configFile}).Error("Unable to parse")
	}

	// initialize FilesDB connection
	FilesDB, err = InitFilesDB()
	defer FilesDB.Close()
	if err != nil {
		logs.WithFields(logs.Fields{"Error": err}).Fatal("FilesDB")
	}

	var templates ServerTemplates
	tmplData := make(map[string]interface{})
	tmplData["Time"] = time.Now()
	tmplData["Version"] = info()
	_top = templates.Top(Config.Templates, tmplData)
	_bottom = templates.Bottom(Config.Templates, tmplData)
	_search = templates.SearchForm(Config.Templates, tmplData)

	// assign handlers
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(Config.Styles))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(Config.Jscripts))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(Config.Images))))
	http.HandleFunc("/", AuthHandler)

	addr := fmt.Sprintf(":%d", Config.Port)
	logs.WithFields(logs.Fields{"Addr": addr}).Info("Starting HTTP server")
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Fatal("ListenAndServe: ")
	}
}
