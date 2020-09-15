package main

// templates module
//
// Copyright (c) 2019 - Valentin Kuznetsov <vkuznet@gmail.com>
//

import (
	"bytes"
	"html/template"
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

// ServerTemplates structure
type ServerTemplates struct {
	top, bottom, pagination, files, searchForm, cards, serverError, keys, zero, record, status string
}

// Top method for ServerTemplates structure
func (q ServerTemplates) Top(tdir string, tmplData map[string]interface{}) string {
	if q.top != "" {
		return q.top
	}
	q.top = parseTmpl(Config.Templates, "top.tmpl", tmplData)
	return q.top
}

// Bottom method for ServerTemplates structure
func (q ServerTemplates) Bottom(tdir string, tmplData map[string]interface{}) string {
	if q.bottom != "" {
		return q.bottom
	}
	q.bottom = parseTmpl(Config.Templates, "bottom.tmpl", tmplData)
	return q.bottom
}

// LoginForm method for ServerTemplates structure
func (q ServerTemplates) LoginForm(tdir string, tmplData map[string]interface{}) string {
	if q.searchForm != "" {
		return q.searchForm
	}
	q.searchForm = parseTmpl(Config.Templates, "login.tmpl", tmplData)
	return q.searchForm
}

// LogoutForm method for ServerTemplates structure
func (q ServerTemplates) LogoutForm(tdir string, tmplData map[string]interface{}) string {
	if q.searchForm != "" {
		return q.searchForm
	}
	q.searchForm = parseTmpl(Config.Templates, "logout.tmpl", tmplData)
	return q.searchForm
}

// SearchForm method for ServerTemplates structure
func (q ServerTemplates) SearchForm(tdir string, tmplData map[string]interface{}) string {
	if q.searchForm != "" {
		return q.searchForm
	}
	q.searchForm = parseTmpl(Config.Templates, "searchform.tmpl", tmplData)
	return q.searchForm
}

// FAQ method for ServerTemplates structure
func (q ServerTemplates) FAQ(tdir string, tmplData map[string]interface{}) string {
	if q.top != "" {
		return q.top
	}
	q.top = parseTmpl(Config.Templates, "faq.tmpl", tmplData)
	return q.top
}

// Record method for ServerTemplates structure
func (q ServerTemplates) Record(tdir string, tmplData map[string]interface{}) string {
	if q.top != "" {
		return q.top
	}
	q.top = parseTmpl(Config.Templates, "record.tmpl", tmplData)
	return q.top
}

// Keys method for ServerTemplates structure
func (q ServerTemplates) Keys(tdir string, tmplData map[string]interface{}) string {
	if q.top != "" {
		return q.top
	}
	q.top = parseTmpl(Config.Templates, "keys.tmpl", tmplData)
	return q.top
}

// Confirm method for ServerTemplates structure
func (q ServerTemplates) Confirm(tdir string, tmplData map[string]interface{}) string {
	if q.top != "" {
		return q.top
	}
	q.top = parseTmpl(Config.Templates, "confirm.tmpl", tmplData)
	return q.top
}

// Pagination  method for ServerTemplates structure
func (q ServerTemplates) Pagination(tdir string, tmplData map[string]interface{}) string {
	if q.pagination != "" {
		return q.pagination
	}
	q.pagination = parseTmpl(Config.Templates, "pagination.tmpl", tmplData)
	return q.pagination
}

// Files  method for ServerTemplates structure
func (q ServerTemplates) Files(tdir string, tmplData map[string]interface{}) string {
	if q.files != "" {
		return q.files
	}
	q.files = parseTmpl(Config.Templates, "files.tmpl", tmplData)
	return q.files
}

// ServerError method for ServerTemplates structure
func (q ServerTemplates) ServerError(tdir string, tmplData map[string]interface{}) string {
	if q.serverError != "" {
		return q.serverError
	}
	q.serverError = parseTmpl(Config.Templates, "error.tmpl", tmplData)
	return q.serverError
}

// ServerZeroResults method for ServerTemplates structure
func (q ServerTemplates) ServerZeroResults(tdir string, tmplData map[string]interface{}) string {
	if q.zero != "" {
		return q.zero
	}
	q.zero = parseTmpl(Config.Templates, "zero_results.tmpl", tmplData)
	return q.zero
}

// Status method for ServerTemplates structure
func (q ServerTemplates) Status(tdir string, tmplData map[string]interface{}) string {
	if q.status != "" {
		return q.status
	}
	q.status = parseTmpl(Config.Templates, "status.tmpl", tmplData)
	return q.status
}

// Update method for ServerTemplates structure
func (q ServerTemplates) Update(tdir string, tmplData map[string]interface{}) string {
	if q.record != "" {
		return q.record
	}
	q.record = parseTmpl(Config.Templates, "update.tmpl", tmplData)
	return q.record
}
