package main

import (
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/kissen/httpstatus"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
)

// GET /
func GetIndex(w http.ResponseWriter, r *http.Request) {
	lease := LockRead()
	defer lease.Unlock()

	fs, err := Files()
	if err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	lease.Unlock()

	vs := map[string]interface{}{
		"Uploads": fs,
	}

	Render(w, r, "index.tmpl", vs)
}

// GET /static/{resource_id}
func GetStatic(w http.ResponseWriter, r *http.Request) {
	box := packr.NewBox("static")
	filename := path.Base(r.URL.Path)

	if !box.Has(filename) {
		DoError(w, r, http.StatusNotFound, "no resource with that name")
		return
	}

	mimetype := mime.TypeByExtension(path.Ext(filename))

	w.Header().Add("Content-Type", mimetype)

	if _, err := w.Write(box.Bytes(filename)); err != nil {
		log.Printf(`serving static filename="%v" failed with err="%v"`, filename, err)
	}
}

// GET /favicon.ico
func GetFavicon(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/paperclip.svg", http.StatusPermanentRedirect)
}

// GET /files/{file_id}/{file_name}
func GetFile(w http.ResponseWriter, r *http.Request) {
	fileId, ok := mux.Vars(r)["file_id"]
	if !ok {
		DoError(w, r, http.StatusBadRequest, "missing file_id")
		return
	}

	fileName, ok := mux.Vars(r)["file_name"]
	if !ok {
		DoError(w, r, http.StatusBadRequest, "missing file_name")
		return
	}

	lease := LockRead()
	defer lease.Unlock()

	fm, err := LoadFile(fileId)
	if err != nil {
		DoError(w, r, http.StatusNotFound, err.Error())
		return
	}

	if fileName != fm.Name {
		DoError(w, r, http.StatusNotFound, "")
		return
	}

	fd, err := os.Open(fm.LocalPath)
	if err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	defer fd.Close()
	lease.Unlock()

	contentType := fm.ContentType
	w.Header().Set("Content-Type", contentType)

	contentLength := strconv.FormatInt(fm.Size, 10)
	w.Header().Set("Content-Length", contentLength)

	if _, err := io.Copy(w, fd); err != nil {
		log.Printf(`serving fileId="%v" failed with err="%v"`, fileId, err)
	}
}

// POST /submit
func PostSubmit(w http.ResponseWriter, r *http.Request) {
	config := GetConfig()

	r.Body = http.MaxBytesReader(w, r.Body, config.MaxFileSize)
	r.ParseMultipartForm(16 * 1024 * 1024)  // 16 MiB buffer

	file, handler, err := r.FormFile("file")
	if err != nil {
		DoError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	lease := LockWrite()
	defer lease.Unlock()

	_, err = CreateFile(file, handler.Filename)
	if err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// POST /delete
func PostDelete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	lease := LockWrite()
	defer lease.Unlock()

	if err := DeleteFile(id); err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
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
		vs := map[string]interface{}{
			"Status":      status,
			"StatusText":  http.StatusText(status),
			"Description": httpstatus.Describe(status),
			"Cause":       message,
		}

		Render(w, r, "error.tmpl", vs)
	}
}
