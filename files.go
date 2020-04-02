package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path"
	"path/filepath"
	"time"
)

type File struct {
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

// Get a listing of all uploaded files.
func Files() (uploads []*File, err error) {
	uploadsDirectory := GetConfig().UploadsDirectory

	LockRead()
	defer UnlockRead()

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
		}

		uploads = append(uploads, upload)
	}

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

	return &meta, nil
}

// Given a reader that contains bytes for a file, store those contents
// as a new file in the storage directory.
//
// Returns the metadata for the created file.
func CreateFile(src io.Reader, filename string) (*File, error) {
	id := uuid.New().String()

	storageDir := GetConfig().UploadsDirectory
	baseDir := filepath.Join(storageDir, id)
	metaPath := filepath.Join(baseDir, "meta.json")
	storagePath := filepath.Join(baseDir, "storage.bin")

	LockWrite()
	defer UnlockWrite()

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
		Name:          id,
		Size:          nbytes,
		UploadedOnUTC: time.Now().UTC(),
		ContentType:   mime.TypeByExtension(path.Ext(filename)),
		LocalPath:     storagePath,
	}

	metabytes, err := json.Marshal(&meta)
	if err != nil {
		return nil, errors.Wrap(err, "error marhaling meta data")
	}

	if err := ioutil.WriteFile(metaPath, metabytes, 400); err != nil {
		return nil, errors.Wrap(err, "error writing meta file")
	}

	return &meta, nil
}
