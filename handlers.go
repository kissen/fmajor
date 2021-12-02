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

// Read out requested files for a request to /files/{file_id}/{file_name}
// and write the headers to the response body. Used in both the GET and HEAD
// requests to /files.
func WriteHeadersForFile(w http.ResponseWriter, r *http.Request) (fm *File, ok bool) {
	// First parse out the requested file id from the request.

	fileId, ok := mux.Vars(r)["file_id"]
	if !ok {
		DoError(w, r, http.StatusBadRequest, "missing file_id")
		return nil, false
	}

	fileName, ok := mux.Vars(r)["file_name"]
	if !ok {
		DoError(w, r, http.StatusBadRequest, "missing file_name")
		return nil, false
	}

	fm, err := LoadFile(fileId)
	if err != nil {
		DoError(w, r, http.StatusNotFound, err.Error())
		return nil, false
	}

	if fileName != fm.Name {
		DoError(w, r, http.StatusNotFound, "")
		return nil, false
	}

	// Set content type header. This is used by browser to determine how
	// a given object is to be displayed (e.g. inline vs download).

	contentType := fm.ContentType
	w.Header().Set("Content-Type", contentType)

	// Set disposition header. This is so only safe content types are shown
	// inline. In particular, we do not want to show HTML files as they would be
	// served like a normal page rather than a download.

	contentDisposition := "attachment"
	if fm.Inline() {
		contentDisposition = "inline"
	}

	w.Header().Set("Content-Disposition", contentDisposition)

	// Set length header. Not really necessary as we aren't keeping the connection
	// open, but it's nice to do so.

	contentLength := strconv.FormatInt(fm.Size, 10)
	w.Header().Set("Content-Length", contentLength)

	// Set last-modified header. Browsers can use this information to determine
	// whether a given object should be re-downloaded.

	rfc113 := "Mon, 02 Jan 2006 15:04:05 MST"
	lastModified := fm.UploadedOnUTC.Format(rfc113)
	w.Header().Set("Last-Modified", lastModified)

	// Set etag header. Similar to the last-modified header, browsers use this
	// unique file id to check with their caches. The etag is an opaque unique
	// id for a given file.  As such we can just use our uuid.

	w.Header().Set("ETag", fm.Id)

	// Success! Return the file for further processing.

	return fm, true
}

// GET /files/{file_id}/{file_name}
func GetFile(w http.ResponseWriter, r *http.Request) {
	var (
		fm  *File
		ok  bool
		fd  *os.File
		err error
	)

	// Aquire the lease so the underlying file does not get to change while we
	// are serving it.

	lease := LockRead()
	defer lease.Unlock()

	// Write out headers.

	if fm, ok = WriteHeadersForFile(w, r); !ok {
		return
	}

	// Open actual file and serve it to the client.

	if fd, err = os.Open(fm.LocalPath); err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	defer fd.Close()
	lease.Unlock()

	if _, err := io.Copy(w, fd); err != nil {
		log.Printf(`serving fileId="%v" failed with err="%v"`, fm.Id, err)
	}
}

// HEAD /files/{file_id}/{file_name}
func HeadFile(w http.ResponseWriter, r *http.Request) {
	// Acquire the lease. Technically there is a race condition between HEAD and
	// GET request, but in practice the worst thing that can happen is that a
	// deleted file will be shown as cached content by the browser.

	lease := LockRead()
	defer lease.Unlock()

	// Really we just have to write out the headers. WriteHeadersForFile handles
	// all errors for us.

	WriteHeadersForFile(w, r)
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
