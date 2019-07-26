package main

import "net/http"

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
