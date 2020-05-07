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
	if ok, _ := IsAuthorized(r); !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

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

// GET /login
func GetLogin(w http.ResponseWriter, r *http.Request) {
	if ok, _ := IsAuthorized(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	Render(w, r, "login.tmpl", nil)
}

// POST /login
func PostLogin(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	if password == "" {
		DoError(w, r, http.StatusBadRequest, "missing password")
		return
	}

	if !IsValidPassword(password) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := SetAuthorized(w); err != nil {
		log.Println(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// POST /logout
func PostLogout(w http.ResponseWriter, r *http.Request) {
	if authed := ErrorIfNotAuthorized(w, r); !authed {
		return
	}

	if err := SetUnauthorized(w); err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	// set content type header

	contentType := fm.ContentType
	w.Header().Set("Content-Type", contentType)

	// set disposition header; this is so only safe content types
	// are shown inline

	contentDisposition := "attachment"
	if fm.Inline() {
		contentDisposition = "inline"
	}

	w.Header().Set("Content-Disposition", contentDisposition)

	// set length header

	contentLength := strconv.FormatInt(fm.Size, 10)
	w.Header().Set("Content-Length", contentLength)

	// write out file

	if _, err := io.Copy(w, fd); err != nil {
		log.Printf(`serving fileId="%v" failed with err="%v"`, fileId, err)
	}
}

// POST /submit
func PostSubmit(w http.ResponseWriter, r *http.Request) {
	if authed := ErrorIfNotAuthorized(w, r); !authed {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, config.MaxFileSize)
	r.ParseMultipartForm(16 * 1024 * 1024) // 16 MiB buffer

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
	if authed := ErrorIfNotAuthorized(w, r); !authed {
		return
	}

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

// Check whether r is an privileged request, i.e. whether it contains
// a valid log in cookie.
//
// If this request is in fact privileged, this function returns true
// and does not touch /he response writer. If the request is not privileged,
// write an error to w and return false.
func ErrorIfNotAuthorized(w http.ResponseWriter, r *http.Request) (authed bool) {
	var err error

	authed, err = IsAuthorized(r)
	if err != nil {
		DoError(w, r, http.StatusUnauthorized, err.Error())
		return false
	}

	if !authed {
		DoError(w, r, http.StatusUnauthorized, "not logged in")
		return false
	}

	return true
}

// Return an error handler for status.
func Error(status int, cause string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vs := map[string]interface{}{
			"Status":      status,
			"StatusText":  http.StatusText(status),
			"Description": httpstatus.Describe(status),
			"Cause":       cause,
		}

		Render(w, r, "error.tmpl", vs)
	}
}
