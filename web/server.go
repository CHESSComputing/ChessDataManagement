package main

// web server module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	logs "github.com/sirupsen/logrus"
	"gopkg.in/jcmturner/gokrb5.v7/keytab"
	"gopkg.in/jcmturner/gokrb5.v7/service"
	"gopkg.in/jcmturner/gokrb5.v7/spnego"

	_ "expvar"         // to be used for monitoring, see https://github.com/divan/expvarmon
	_ "net/http/pprof" // profiler, see https://golang.org/pkg/net/http/pprof/
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
	if Config.Verbose > 0 {
		logs.SetLevel(logs.DebugLevel)
	}
	if err != nil {
		logs.WithFields(logs.Fields{"Time": time.Now(), "Config": configFile}).Error("Unable to parse")
	}
	fmt.Println("Configuration:", Config.String())

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

	// configure kerberos auth
	kt, err := keytab.Load(Config.Keytab)
	l := log.New(os.Stderr, "GOKRB5 Service: ", log.Ldate|log.Ltime|log.Lshortfile)
	h := http.HandlerFunc(LoginHandler)
	http.Handle("/login", spnego.SPNEGOKRB5Authenticate(h, kt, service.Logger(l)))

	// assign handlers
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(Config.Styles))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(Config.Jscripts))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(Config.Images))))
	http.HandleFunc("/auth", KAuthHandler)
	http.HandleFunc("/api", ApiHandler)
	http.HandleFunc("/search", SearchHandler)
	http.HandleFunc("/", AuthHandler)

	// Start server
	addr := fmt.Sprintf(":%d", Config.Port)
	_, e1 := os.Stat(Config.ServerCrt)
	_, e2 := os.Stat(Config.ServerKey)
	if e1 == nil && e2 == nil {
		//start HTTPS server which require user certificates
		server := &http.Server{
			Addr: addr,
			TLSConfig: &tls.Config{
				ClientAuth: tls.RequestClientCert,
			},
		}
		logs.WithFields(logs.Fields{"Addr": addr}).Info("Starting HTTPs server")
		err = server.ListenAndServeTLS(Config.ServerCrt, Config.ServerKey)
	} else {
		// Start server without user certificates
		logs.WithFields(logs.Fields{"Addr": addr}).Info("Starting HTTP server")
		err = http.ListenAndServe(addr, nil)
	}
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Fatal("ListenAndServe: ")
	}
}
