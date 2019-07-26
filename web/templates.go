package main

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
	top, bottom, searchForm, cards, dasError, dasKeys, dasZero string
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

// SearchForm method for ServerTemplates structure
func (q ServerTemplates) SearchForm(tdir string, tmplData map[string]interface{}) string {
	if q.searchForm != "" {
		return q.searchForm
	}
	q.searchForm = parseTmpl(Config.Templates, "searchform.tmpl", tmplData)
	return q.searchForm
}

// Cards method for ServerTemplates structure
func (q ServerTemplates) Cards(tdir string, tmplData map[string]interface{}) string {
	tmplData["CardsClass"] = "hide"
	if q.cards != "" {
		return q.cards
	}
	q.cards = parseTmpl(Config.Templates, "cards.tmpl", tmplData)
	return q.cards
}

// FAQ method for ServerTemplates structure
func (q ServerTemplates) FAQ(tdir string, tmplData map[string]interface{}) string {
	if q.top != "" {
		return q.top
	}
	q.top = parseTmpl(Config.Templates, "faq.tmpl", tmplData)
	return q.top
}

// ApiRecord method for ServerTemplates structure
func (q ServerTemplates) ApiRecord(tdir string, tmplData map[string]interface{}) string {
	if q.top != "" {
		return q.top
	}
	q.top = parseTmpl(Config.Templates, "api_record.tmpl", tmplData)
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

// Services method for ServerTemplates structure
func (q ServerTemplates) Services(tdir string, tmplData map[string]interface{}) string {
	if q.top != "" {
		return q.top
	}
	q.top = parseTmpl(Config.Templates, "services.tmpl", tmplData)
	return q.top
}

// Pagination  method for ServerTemplates structure
func (q ServerTemplates) Pagination(tdir string, tmplData map[string]interface{}) string {
	if q.searchForm != "" {
		return q.searchForm
	}
	q.searchForm = parseTmpl(Config.Templates, "pagination.tmpl", tmplData)
	return q.searchForm
}

// ServerError method for ServerTemplates structure
func (q ServerTemplates) ServerError(tdir string, tmplData map[string]interface{}) string {
	if q.dasError != "" {
		return q.dasError
	}
	q.dasError = parseTmpl(Config.Templates, "error.tmpl", tmplData)
	return q.dasError
}

// ServerZeroResults method for ServerTemplates structure
func (q ServerTemplates) ServerZeroResults(tdir string, tmplData map[string]interface{}) string {
	if q.dasZero != "" {
		return q.dasZero
	}
	q.dasZero = parseTmpl(Config.Templates, "zero_results.tmpl", tmplData)
	return q.dasZero
}

// Status method for ServerTemplates structure
func (q ServerTemplates) Status(tdir string, tmplData map[string]interface{}) string {
	if q.dasError != "" {
		return q.dasError
	}
	q.dasError = parseTmpl(Config.Templates, "status.tmpl", tmplData)
	return q.dasError
}
