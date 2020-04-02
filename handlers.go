package main

import (
	"fmt"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/kissen/httpstatus"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
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
		DoError(w, r, http.StatusNotFound, "no resource with that name")
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
	fileId, ok := mux.Vars(r)["file_id"]
	if !ok {
		DoError(w, r, http.StatusBadRequest, "missing file_id")
		return
	}

	LockRead()
	defer UnlockRead()

	fm, err := LoadFile(fileId)
	if err != nil {
		DoError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	fd, err := os.Open(fm.LocalPath)
	if err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	defer fd.Close()

	if _, err := io.Copy(w, fd); err != nil {
		log.Printf(`err="%v" for fileId="%v"`, err, fileId)
	}
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

	LockWrite()
	defer UnlockWrite()

	_, err = CreateFile(file, handler.Filename)
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func DoError(w http.ResponseWriter, r *http.Request, status int, message string) {
	Error(status, message).ServeHTTP(w, r)
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
