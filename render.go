package main

import (
	"net/http"

	"github.com/kissen/fmajor/templates"
)

// Render out page to (w, r) with parameters vs.
//
// If this function encounters errors, it calls http.Error to forward
// that error to the client. It does not use our fancy Error function
// because that function uses Render and I would prefer to avoid infinite
// recursion.
func Render(w http.ResponseWriter, r *http.Request, page string, vs map[string]interface{}) {
	// set some flags used by all templates

	if vs == nil {
		vs = make(map[string]interface{})
	}

	if authed, err := IsAuthorized(r); err == nil {
		vs["IsAuthorized"] = authed
	}

	// render out the page to the HTTP writer

	if err := templates.Render(w, r, page, vs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
