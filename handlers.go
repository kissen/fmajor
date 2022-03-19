package main

import (
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/kissen/fmajor/static"
	"github.com/kissen/httpstatus"
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
	var filename string
	var bytes []byte
	var err error

	// Open the file.

	if filename = path.Base(r.URL.Path); filename == "" {
		DoError(w, r, http.StatusBadRequest, "empty filename")
		return
	}

	if bytes, err = static.ReadFile(filename); err != nil {
		DoError(w, r, http.StatusNotFound, "no resource with that name")
		return
	}

	// Set mime type.

	mimetype := mime.TypeByExtension(path.Ext(filename))
	w.Header().Add("Content-Type", mimetype)

	// Send out file.

	if _, err := w.Write(bytes); err != nil {
		log.Printf(`serving static filename="%v" failed with err="%v"`, filename, err)
	}
}

// GET /favicon.ico
func GetFavicon(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/paperclip.svg", http.StatusPermanentRedirect)
}

func WriteHeadersFor(fm *File, w http.ResponseWriter) {
	contentType := fm.ContentType
	inline := fm.Inline()
	size := fm.Size
	lastModified := fm.UploadedOnUTC
	etag := fm.Id

	WriteHeadersTo(w, contentType, etag, inline, lastModified, &size)
}

func WriteThumbnailHeadersFor(fm *File, w http.ResponseWriter) {
	contentType := "image/jpeg"
	inline := true
	size := fm.ThumbnailSize
	lastModified := fm.UploadedOnUTC
	etag := fmt.Sprintf("%v-thumbnail", fm.Id)

	WriteHeadersTo(w, contentType, etag, inline, lastModified, size)
}

func WriteHeadersTo(w http.ResponseWriter, contentType, etag string, inline bool, lastModified time.Time, size *int64) {
	w.Header().Set("Cache-Control", "max-age=15552000")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("ETag", etag)

	if inline {
		w.Header().Set("Content-Disposition", "inline")
	} else {
		w.Header().Set("Content-Disposition", "attachment")
	}

	if size != nil {
		contentLength := strconv.FormatInt(*size, 10)
		w.Header().Set("Content-Length", contentLength)
	}

	rfc113 := "Mon, 02 Jan 2006 15:04:05 GMT"
	lastModifiedString := lastModified.Format(rfc113)
	w.Header().Set("Last-Modified", lastModifiedString)
}

// GET /files/{file_id}/{file_name}
func GetFile(w http.ResponseWriter, r *http.Request) {
	DoFile(w, r, true)
}

// HEAD /files/{file_id}/{file_name}
func HeadFile(w http.ResponseWriter, r *http.Request) {
	DoFile(w, r, false)
}

// GET/HEAD /files/{file_id}/{file_name}
func DoFile(w http.ResponseWriter, r *http.Request, doSendBody bool) {
	var (
		err      error
		fd       *os.File
		fileId   string
		fileName string
		fm       *File
		ok       bool
	)

	// Write out headers.

	if fileId, ok = mux.Vars(r)["file_id"]; !ok {
		DoError(w, r, http.StatusBadRequest, "missing file_id")
		return
	}

	if fileName, ok = mux.Vars(r)["file_name"]; !ok {
		DoError(w, r, http.StatusBadRequest, "missing file_name")
		return
	}

	// Aquire the lease so the underlying file does not get to change while we
	// are serving it.

	lease := LockRead()
	defer lease.Unlock()

	// Get meta data struct and write out the respective headers to the client.

	if fm, err = LoadFile(fileId); err != nil {
		DoError(w, r, http.StatusNotFound, err.Error())
		return
	}

	if fileName != fm.Name {
		DoError(w, r, http.StatusNotFound, "bad file name")
		return
	}

	WriteHeadersFor(fm, w)

	// If we are only serving a HEAD request, we really only have to care about
	// the headers.

	if !doSendBody {
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

// GET /f/{short_id}
func GetShort(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		meta    *File
		ok      bool
		shortId string
	)

	if shortId, ok = mux.Vars(r)["short_id"]; !ok {
		DoError(w, r, http.StatusBadRequest, "missing short_id")
		return
	}

	if meta, err = LoadFile(shortId); err != nil {
		DoError(w, r, http.StatusNotFound, err.Error())
		return
	}

	full := path.Join("/", "files", meta.Id, meta.Name)
	http.Redirect(w, r, full, http.StatusMovedPermanently)
}

// GET /thumbnails/{file_id}/thumbnail.jpg
func GetThumbnail(w http.ResponseWriter, r *http.Request) {
	DoThumbnail(w, r, true)
}

// HEAD /thumbnails/{file_id}/thumbnail.jpg
func HeadThumbnails(w http.ResponseWriter, r *http.Request) {
	DoThumbnail(w, r, false)
}

// GET/HEAD /thumbnails/{file_id}/thumbnail.jpg
func DoThumbnail(w http.ResponseWriter, r *http.Request, doSendBody bool) {
	var (
		err    error
		fd     *os.File
		fileId string
		fm     *File
		ok     bool
	)

	if fileId, ok = mux.Vars(r)["file_id"]; !ok {
		DoError(w, r, http.StatusBadRequest, "missing file_id")
		return
	}

	lease := LockRead()
	defer lease.Unlock()

	if fm, err = LoadFile(fileId); err != nil {
		DoError(w, r, http.StatusNotFound, err.Error())
		return
	}

	if fm.ThumbnailPath == nil {
		DoError(w, r, http.StatusNotFound, "no thumbnail for given file")
		return
	}

	WriteThumbnailHeadersFor(fm, w)

	if !doSendBody {
		return
	}

	if fd, err = os.Open(*fm.ThumbnailPath); err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	defer fd.Close()
	lease.Unlock()

	if _, err = io.Copy(w, fd); err != nil {
		log.Printf(`serving thumbnail for fileId="%v" failed with err="%v"`, fm.Id, err)
	}
}

// POST /submit
func PostSubmit(w http.ResponseWriter, r *http.Request) {
	var (
		createShortId bool
		err           error
		file          multipart.File
		header        *multipart.FileHeader
	)

	// Only allow users to upload.

	if authed := ErrorIfNotAuthorized(w, r); !authed {
		return
	}

	// Get file contents.

	r.Body = http.MaxBytesReader(w, r.Body, config.MaxFileSize)
	r.ParseMultipartForm(16 * 1024 * 1024) // 16 MiB buffer

	if file, header, err = r.FormFile("file"); err != nil {
		DoError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	defer file.Close()

	// Register file in bookkeeping.

	if value := r.FormValue("create_short_id"); value == "true" {
		createShortId = true
	}

	lease := LockWrite()
	defer lease.Unlock()

	if _, err = CreateFile(file, header.Filename, createShortId); err != nil {
		DoError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Forward to index page.

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
