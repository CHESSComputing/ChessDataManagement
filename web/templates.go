package main

// templates module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"bytes"
	"html/template"
	"log"
	"path/filepath"
)

// consume list of templates and release their full path counterparts
func fileNames(tdir string, filenames ...string) []string {
	flist := []string{}
	for _, fname := range filenames {
		flist = append(flist, filepath.Join(tdir, fname))
	}
	return flist
}

// parse template with given data
func parseTmpl(tdir, tmpl string, data interface{}) string {
	buf := new(bytes.Buffer)
	filenames := fileNames(tdir, tmpl)
	funcMap := template.FuncMap{
		// The name "oddFunc" is what the function will be called in the template text.
		"oddFunc": func(i int) bool {
			if i%2 == 0 {
				return true
			}
			return false
		},
		// The name "inListFunc" is what the function will be called in the template text.
		"inListFunc": func(a string, list []string) bool {
			check := 0
			for _, b := range list {
				if b == a {
					check++
				}
			}
			if check != 0 {
				return true
			}
			return false
		},
	}
	t := template.Must(template.New(tmpl).Funcs(funcMap).ParseFiles(filenames...))
	err := t.Execute(buf, data)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

// Templates structure
type Templates struct {
	html string
}

// Tmpl method for ServerTemplates structure
func (q Templates) Tmpl(tdir, tfile string, tmplData map[string]interface{}) string {
	if q.html != "" {
		return q.html
	}
	if Config.Verbose > 0 {
		log.Println("template.Tmpl load", tdir, tfile)
	}
	q.html = parseTmpl(tdir, tfile, tmplData)
	return q.html
}
