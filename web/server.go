package main

// web server module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"gopkg.in/jcmturner/gokrb5.v7/keytab"
	"gopkg.in/jcmturner/gokrb5.v7/service"
	"gopkg.in/jcmturner/gokrb5.v7/spnego"

	_ "expvar"         // to be used for monitoring, see https://github.com/divan/expvarmon
	_ "net/http/pprof" // profiler, see https://golang.org/pkg/net/http/pprof/

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// global variables
var _top, _bottom, _search string
var _beamlines []string
var _smgr SchemaManager

// Time0 represents initial time when we started the server
var Time0 time.Time

// custom rotate logger
type rotateLogWriter struct {
	RotateLogs *rotatelogs.RotateLogs
}

func (w rotateLogWriter) Write(data []byte) (int, error) {
	return w.RotateLogs.Write([]byte(utcMsg(data)))
}

// helper function to use proper UTC message in a logger
func utcMsg(data []byte) string {
	s := string(data)
	v, e := url.QueryUnescape(s)
	if e == nil {
		return v
	}
	return s
}

// Server code
func Server(configFile string) {
	Time0 = time.Now()
	var err error
	ParseConfig(configFile)
	// set log file or log output
	if Config.LogFile != "" {
		logName := Config.LogFile + "-%Y%m%d"
		hostname, err := os.Hostname()
		if err == nil {
			logName = Config.LogFile + "-" + hostname + "-%Y%m%d"
		}
		rl, err := rotatelogs.New(logName)
		if err == nil {
			rotlogs := rotateLogWriter{RotateLogs: rl}
			log.SetOutput(rotlogs)
		} else {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		}
	} else {
		// log time, filename, and line number
		if Config.Verbose > 0 {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		} else {
			log.SetFlags(log.LstdFlags)
		}
	}
	if len(Config.SchemaFiles) == 0 {
		log.Fatal("Configuration does not have schema files")
	}

	log.Println("Configuration:", Config.String())
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// initialize FilesDB connection
	FilesDB, err = InitFilesDB()
	defer FilesDB.Close()
	if err != nil {
		log.Printf("FilesDB error: %v\n", err)
	}
	// initialize schema manager
	_smgr = SchemaManager{}
	for _, fname := range Config.SchemaFiles {
		_, err := _smgr.Load(fname)
		if err != nil {
			log.Fatalf("unable to load %s error %v", fname, err)
		}
		_beamlines = append(_beamlines, fileName(fname))
	}
	log.Println("Schema", _smgr.String())

	var templates Templates
	tmplData := make(map[string]interface{})
	tmplData["Time"] = time.Now()
	tmplData["Version"] = info()
	_top = templates.Tmpl(Config.Templates, "top.tmpl", tmplData)
	_bottom = templates.Tmpl(Config.Templates, "bottom.tmpl", tmplData)
	_search = templates.Tmpl(Config.Templates, "searchform.tmpl", tmplData)

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
	http.HandleFunc("/api", APIHandler)
	http.HandleFunc("/search", SearchHandler)
	http.HandleFunc("/files", FilesHandler)
	http.HandleFunc("/", AuthHandler)

	// Start server
	addr := fmt.Sprintf(":%d", Config.Port)
	_, e1 := os.Stat(Config.ServerCrt)
	_, e2 := os.Stat(Config.ServerKey)
	if e1 == nil && e2 == nil {
		//start HTTPS server which require user certificates
		rootCA := x509.NewCertPool()
		caCert, _ := ioutil.ReadFile(Config.RootCA)
		rootCA.AppendCertsFromPEM(caCert)
		server := &http.Server{
			Addr: addr,
			TLSConfig: &tls.Config{
				//                 ClientAuth: tls.RequestClientCert,
				RootCAs: rootCA,
			},
		}
		log.Printf("Starting HTTPs server, %v\n", addr)
		err = server.ListenAndServeTLS(Config.ServerCrt, Config.ServerKey)
	} else {
		// Start server without user certificates
		log.Printf("Starting HTTP server, %v\n", addr)
		err = http.ListenAndServe(addr, nil)
	}
	if err != nil {
		log.Fatalf("Unable to start server, %v\n", err)
	}
}
