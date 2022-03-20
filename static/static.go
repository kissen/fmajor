package static

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"path"
)

// go:embed css/*
var css embed.FS

// go:embed fonts/*
var fonts embed.FS

// go:embed js/*
var js embed.FS

// go:embed svg/*
var svg embed.FS

func readFile(storage fs.FS, path string) (reader io.Reader, fi fs.FileInfo, err error) {
	var (
		bs     []byte
		handle fs.File
	)

	if handle, err = storage.Open(path); err != nil {
		return nil, nil, err
	}

	defer handle.Close()

	if fi, err = handle.Stat(); err != nil {
		return nil, nil, err
	}

	if bs, err = ioutil.ReadAll(handle); err != nil {
		return nil, nil, err
	}

	return bytes.NewReader(bs), fi, nil
}

// Try to read file with given name in any of the /static directories.
func ReadFile(filename string) (io.Reader, fs.FileInfo, error) {
	choices := map[string]embed.FS{
		"css": css, "fonts": fonts, "js": js, "svg": svg,
	}

	for name, dir := range choices {
		fullPath := path.Join(name, filename)

		if reader, fi, err := readFile(dir, fullPath); err == nil {
			return reader, fi, nil
		}
	}

	return nil, nil, errors.New("no such file or directory")
}
