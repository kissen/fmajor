package main

import (
	"fmt"
	"github.com/gobuffalo/packr"
	"github.com/kissen/httpstatus"
	"log"
	"mime"
	"net/http"
	"path"
)

// GET /
func GetIndex(w http.ResponseWriter, r *http.Request) {
	vs := map[string]string{
		"Title": "FMajor File Hosting",
	}

	Render(w, r, "index.tmpl", vs)
}

// GET /static/{resourceid}
func GetStatic(w http.ResponseWriter, r *http.Request) {
	box := packr.NewBox("static")
	filename := path.Base(r.URL.Path)

	if !box.Has(filename) {
		Error(http.StatusNotFound, "no resource with that name")
		return
	}

	mimetype := mime.TypeByExtension(path.Ext(filename))

	w.Header().Add("Content-Type", mimetype)
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(box.Bytes(filename)); err != nil {
		log.Printf(`writing static filename="%v" failed with err="%v"`, filename, err)
	}
}

// GET /files/{fileid}
func GetFile(w http.ResponseWriter, r *http.Request) {
	Error(http.StatusNotImplemented, "sorry, I'm busy")
}

// POST /submit
func PostSubmit(w http.ResponseWriter, r *http.Request) {
	config := GetConfig()

	r.ParseMultipartForm(config.MaxFileSize)

	file, handler, err := r.FormFile("file")
	if err != nil {
		Error(http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	_, err = CreateFile(file, handler.Filename)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())
		return
	}

	GetIndex(w, r)
}

// Return an error handler for status.
func Error(status int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf(`encountered error with message="%v" status="%v" for URL="%v"`, message, status, r.URL)

		vs := map[string]string{
			"Status":      fmt.Sprintf("%d", status),
			"StatusText":  http.StatusText(status),
			"Description": httpstatus.Describe(status),
			"Cause":       message,
		}

		Render(w, r, "error.tmpl", vs)
	}
}
