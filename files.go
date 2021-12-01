package main

import (
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/kissen/stringset"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// Contains all mime types for which File.Inline should return
// true, that is those mime types that should be shown inline (in
// the web browser). Initialized in init().
var inlineMimeTypes stringset.StringSet

func init() {
	// Initalize inlineMimeTypes. Based on a list of common MIME described types on
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types

	inlineMimeTypes = stringset.NewWith(
		"audio/aac", "image/bmp", "text/css", "text/csv",
		"application/epub+zip", "image/gif", "image/jpeg",
		"text/javascript", "application/json",
		"application/ld+json", "audio/midi", "audio/x-midi",
		"text/javascript", "audio/mpeg", "video/mpeg",
		"audio/ogg", "video/ogg", "application/ogg",
		"audio/opus", "font/otf", "image/png",
		"application/pdf", "image/svg+xml",
		"application/x-shockwave-flash", "image/tiff",
		"video/mp2t", "text/plain", "audio/wav", "audio/webm",
		"video/webm", "image/webp", "application/xml",
	)
}

type File struct {
	// The Id of this file. It is the randomly chosen
	// Id picked by fmajor.
	Id string

	// The filename of the file as reported to the user.
	Name string

	// The size in bytes for reporting to the user.
	Size int64

	// When the file was originally uploaded.
	UploadedOnUTC time.Time

	// An infered content type for this file.
	ContentType string

	// The filepath on the local machine.
	LocalPath string
}

// Return whether any of the fields are set to their zero-value.
// This usually indicates some unmarshal eror.
func (f *File) HasZero() bool {
	if f.Id == "" {
		return true
	}

	if f.Name == "" {
		return true
	}

	if f.Size == 0 {
		return true
	}

	if f.ContentType == "" {
		return true
	}

	if f.LocalPath == "" {
		return true
	}

	return false
}

// Return Size as a human-readable string.
func (f *File) HumanSize() string {
	if f.Size < 0 {
		// shouldn't happen, but who knows...
		return strconv.FormatInt(f.Size, 10)
	} else {
		return humanize.IBytes(uint64(f.Size))
	}
}

// Return whether the file should be inlined, that is shown as-is
// in the browser. This makes sense for images and simple text files.
func (f *File) Inline() bool {
	return inlineMimeTypes.Contains(f.ContentType)
}

// Return upload timestamp as human-readable string.
func (f *File) HumanUploadedOn() string {
	return f.UploadedOnUTC.Format("2006-01-02 15:04")
}

// Get a listing of all uploaded files.
//
// Only call this function if you are holding the global read lock.
func Files() (uploads []*File, err error) {
	uploadsDirectory := GetConfig().UploadsDirectory

	fis, err := ioutil.ReadDir(uploadsDirectory)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}

		id := fi.Name()

		if upload, err := LoadFile(id); err != nil {
			log.Printf("problem while creating file listing: %v", err)
		} else {
			uploads = append(uploads, upload)
		}
	}

	sort.Slice(uploads, func(i, j int) bool {
		return uploads[i].UploadedOnUTC.After(uploads[j].UploadedOnUTC)
	})

	return
}

// Load the metadata for a previously uploaded file.
//
// Only call this function if you are holding the global read lock.
func LoadFile(id string) (*File, error) {
	storageDir := GetConfig().UploadsDirectory
	baseDir := filepath.Join(storageDir, id)
	metaPath := filepath.Join(baseDir, "meta.json")

	metabytes, err := ioutil.ReadFile(metaPath)
	if err != nil {
		return nil, errors.Wrapf(err, `cannot open meta.json for id="%v"`, id)
	}

	var meta File
	if err := json.Unmarshal(metabytes, &meta); err != nil {
		return nil, errors.Wrapf(err, `cannot parse meta.json for id="%v"`, id)
	}

	if meta.HasZero() {
		return nil, fmt.Errorf(`meta.json for id="%v" contains invalid values`, id)
	}

	return &meta, nil
}

// Given a reader that contains bytes for a file, store those contents
// as a new file in the storage directory. Returns the metadata for the
// created file.
//
// Only call this function if you are holding the global write lock.
func CreateFile(src io.Reader, filename string) (*File, error) {
	id := uuid.New().String()

	storageDir := GetConfig().UploadsDirectory
	baseDir := filepath.Join(storageDir, id)
	metaPath := filepath.Join(baseDir, "meta.json")
	storagePath := filepath.Join(baseDir, "storage.bin")

	if err := os.Mkdir(baseDir, 0700); err != nil {
		return nil, errors.Wrapf(err, `cannot create directory for filename="%v"`, filename)
	}

	fd, err := os.Create(storagePath)
	if err != nil {
		DeleteFileAsync(id)
		return nil, errors.Wrapf(err, `cannot create storage.bin for id="%v" filename="%v"`, id, filename)
	}

	defer fd.Close()

	nbytes, err := io.Copy(fd, src)
	if err != nil {
		DeleteFileAsync(id)
		return nil, errors.Wrapf(err, `cannot write storage.bin for id="%v" filename="%v"`, id, filename)
	}

	meta := File{
		Id:            id,
		Name:          filename,
		Size:          nbytes,
		UploadedOnUTC: time.Now().UTC(),
		ContentType:   mime.TypeByExtension(path.Ext(filename)),
		LocalPath:     storagePath,
	}

	if meta.ContentType == "" {
		meta.ContentType = "application/octet-stream"
	}

	metabytes, err := json.Marshal(&meta)
	if err != nil {
		DeleteFileAsync(id)
		return nil, errors.Wrapf(err, `cannot construct meta.json for id="%v" filename="%v"`, id, filename)
	}

	if err := ioutil.WriteFile(metaPath, metabytes, 400); err != nil {
		DeleteFileAsync(id)
		return nil, errors.Wrapf(err, `cannot write meta.json for id="%v" filename="%v"`, id, filename)
	}

	if meta.HasZero() {
		DeleteFileAsync(id)
		return nil, fmt.Errorf(`meta.json for id="%v" filename="%v" contains invalid values`, id, filename)
	}

	return &meta, nil
}

// Delete the file with id from the file system.
//
// Only call this function if you are holding the global write lock.
func DeleteFile(id string) error {
	storageDir := GetConfig().UploadsDirectory
	baseDir := filepath.Join(storageDir, id)
	metaPath := filepath.Join(baseDir, "meta.json")
	storagePath := filepath.Join(baseDir, "storage.bin")

	metaErr := os.Remove(metaPath)
	storageErr := os.Remove(storagePath)
	rmdirErr := os.Remove(baseDir)

	if metaErr != nil {
		return metaErr
	}

	if storageErr != nil {
		return storageErr
	}

	if rmdirErr != nil {
		return rmdirErr
	}

	return nil
}

// In a new goroutine, acquire the write lock and try our best to
// delete the directory for the file with given id.
//
// This function is handy when you (probably) created a broken
// file upload and want to clean up everything it left behind.
// In those cases, call DeleteFileAsync and whatever the upload
// left behind will be deleted when convenient.
//
// This function does not return an error, rather it writes a
// log message when it cannot delete the directory for given id.
func DeleteFileAsync(id string) {
	go func() {
		lease := LockWrite()
		defer lease.Unlock()

		// the following logic is similar to DeleteFile; unlike
		// DeleteFile, we don't care if the file directory is
		// corrupt in some way or missing files, all we care is
		// that we get rid of it

		storageDir := GetConfig().UploadsDirectory
		baseDir := filepath.Join(storageDir, id)
		metaPath := filepath.Join(baseDir, "meta.json")
		storagePath := filepath.Join(baseDir, "storage.bin")

		os.Remove(metaPath)
		os.Remove(storagePath)
		err := os.Remove(baseDir)

		if err != nil {
			log.Printf(`could not clean up id="%v"`, id)
		}
	}()
}
