package main

import (
	"github.com/gobuffalo/packr"
	"html/template"
	"net/http"
)

// Render out page to (w, r) with parameters vs.
//
// If this function encounters errors, it calls http.Error to forward
// that error to the client. It does not use our fancy Error function
// because that function uses Render and I would prefer to avoid infinite
// recursion.
func Render(w http.ResponseWriter, r *http.Request, page string, vs map[string]interface{}) {
	box := packr.NewBox("resources")

	ts, err := template.New("base").Parse(box.String("base.tmpl"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := ts.New("page").Parse(box.String(page)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := ts.Execute(w, vs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
