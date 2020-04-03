package main

import (
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
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

// Return whether any of the fields are set to the zero-value.
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

// Return upload timestamp as human-readable string.
func (f *File) HumanUploadedOn() string {
	return f.UploadedOnUTC.Format("2006/01/02 15:04")
}

// The extension of the original filename.
func (f *File) Ext() string {
	return path.Ext(f.Name)
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

		upload, err := LoadFile(id)
		if err != nil {
			log.Printf(`err="%v" for id="%v"`, err, id)
		} else {
			uploads = append(uploads, upload)
		}
	}

	sort.Slice(uploads, func(i, j int) bool {
		return uploads[i].UploadedOnUTC.After(uploads[j].UploadedOnUTC)
	})

	return
}

// Load the metadata for a previously uploaded.
//
// Only call this function if you are holding the global read lock.
func LoadFile(id string) (*File, error) {
	storageDir := GetConfig().UploadsDirectory
	baseDir := filepath.Join(storageDir, id)
	metaPath := filepath.Join(baseDir, "meta.json")

	metabytes, err := ioutil.ReadFile(metaPath)
	if err != nil {
		return nil, errors.Wrap(err, "corrupt metadata file")
	}

	var meta File
	if err := json.Unmarshal(metabytes, &meta); err != nil {
		return nil, errors.Wrap(err, "corrupt metadata contents")
	}

	if meta.HasZero() {
		return nil, fmt.Errorf(`meta data at metaPath="%v" corrupt`, metaPath)
	}

	return &meta, nil
}

// Given a reader that contains bytes for a file, store those contents
// as a new file in the storage directory. Returns the metadata for the
//created file.
//
// Only call this function if you are holding the global write lock.
func CreateFile(src io.Reader, filename string) (*File, error) {
	id := uuid.New().String()

	storageDir := GetConfig().UploadsDirectory
	baseDir := filepath.Join(storageDir, id)
	metaPath := filepath.Join(baseDir, "meta.json")
	storagePath := filepath.Join(baseDir, "storage.bin")

	if err := os.Mkdir(baseDir, 0700); err != nil {
		return nil, errors.Wrap(err, "error creating directory")
	}

	fd, err := os.Create(storagePath)
	if err != nil {
		return nil, errors.Wrap(err, "error creating storage file")
	}

	defer fd.Close()

	nbytes, err := io.Copy(fd, src)
	if err != nil {
		return nil, errors.Wrap(err, "error writing storage file")
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
		return nil, errors.Wrap(err, "error marhaling meta data")
	}

	if err := ioutil.WriteFile(metaPath, metabytes, 400); err != nil {
		return nil, errors.Wrap(err, "error writing meta file")
	}

	if meta.HasZero() {
		return nil, fmt.Errorf(`create corrupt meta file for id="%v" filename="%v"`, id, filename)
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
