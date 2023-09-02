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
	"strings"
	"time"

	"gopkg.in/jcmturner/gokrb5.v7/keytab"
	"gopkg.in/jcmturner/gokrb5.v7/service"
	"gopkg.in/jcmturner/gokrb5.v7/spnego"

	_ "expvar"         // to be used for monitoring, see https://github.com/divan/expvarmon
	_ "net/http/pprof" // profiler, see https://golang.org/pkg/net/http/pprof/

	"github.com/gorilla/mux"
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

// loggerMiddleware helper function
// https://www.thecodersstop.com/golang/simple-http-request-logging-middleware-in-go/
func loggerMiddleware(r *mux.Router) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				log.Printf(
					"[%s] %s %s %s",
					req.Method,
					req.Host,
					req.URL.Path,
					req.URL.RawQuery,
				)
			}()
			next.ServeHTTP(w, req)
		})
	}
}

// helper function to handle base path of URL requests
func basePath(api string) string {
	base := Config.Base
	if base != "" {
		if strings.HasPrefix(api, "/") {
			api = strings.Replace(api, "/", "", 1)
		}
		if strings.HasPrefix(base, "/") {
			return fmt.Sprintf("%s/%s", base, api)
		}
		return fmt.Sprintf("/%s/%s", base, api)
	}
	return api
}

// Handlers provides helper function to setup all HTTP routes
func Handlers() *mux.Router {
	router := mux.NewRouter()
	router.StrictSlash(true) // to allow /route and /route/ end-points
	router.HandleFunc(basePath("/auth"), KAuthHandler).Methods("GET", "POST")
	router.HandleFunc(basePath("/api"), APIHandler).Methods("POST")
	router.HandleFunc(basePath("/search"), SearchHandler).Methods("GET", "POST")
	router.HandleFunc(basePath("/files"), FilesHandler).Methods("GET", "POST")
	router.HandleFunc(basePath("/faq"), FAQHandler)
	router.HandleFunc(basePath("/status"), StatusHandler)
	router.HandleFunc(basePath("/schemas"), SchemasHandler)
	router.HandleFunc(basePath("/server"), SettingsHandler)
	router.HandleFunc(basePath("/data"), DataHandler)
	router.HandleFunc(basePath("/process"), ProcessHandler)
	router.HandleFunc(basePath("/updateRecord"), UpdateRecordHandler)
	router.HandleFunc(basePath("/json"), JsonHandler)
	router.HandleFunc(basePath("/"), AuthHandler).Methods("GET", "POST")

	// common middleware
	router.Use(loggerMiddleware(router))
	return router
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

	// set log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// dump server configuration
	log.Printf("Configuration:\n%s", Config.String())

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
	tmplData := makeTmplData()
	tmplData["Time"] = time.Now()
	tmplData["Version"] = info()
	_top = templates.Tmpl(Config.Templates, "top.tmpl", tmplData)
	_bottom = templates.Tmpl(Config.Templates, "bottom.tmpl", tmplData)
	_search = templates.Tmpl(Config.Templates, "searchform.tmpl", tmplData)

	// configure kerberos auth
	kt, err := keytab.Load(Config.Keytab)
	l := log.New(os.Stderr, "GOKRB5 Service: ", log.Ldate|log.Ltime|log.Lshortfile)
	h := http.HandlerFunc(LoginHandler)
	http.Handle(basePath("/login"), spnego.SPNEGOKRB5Authenticate(h, kt, service.Logger(l)))

	// assign handlers
	http.Handle(
		basePath("/css/"),
		http.StripPrefix(basePath("/css/"), http.FileServer(http.Dir(Config.Styles))))
	http.Handle(
		basePath("/js/"),
		http.StripPrefix(basePath("/js/"), http.FileServer(http.Dir(Config.Jscripts))))
	http.Handle(
		basePath("/images/"),
		http.StripPrefix(basePath("/images/"), http.FileServer(http.Dir(Config.Images))))
	http.Handle(basePath("/"), Handlers())

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
