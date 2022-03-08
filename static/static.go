package static

import (
	"embed"
	"errors"
	"io/fs"
	"path"
)

//go:embed css/*
var css embed.FS

//go:embed fonts/*
var fonts embed.FS

//go:embed js/*
var js embed.FS

//go:embed svg/*
var svg embed.FS

// Try to read file with given name in any of the /static directories.
func ReadFile(filename string) ([]byte, error) {
	choices := map[string]embed.FS{
		"css": css, "fonts": fonts, "js": js, "svg": svg,
	}

	for name, dir := range choices {
		fullPath := path.Join(name, filename)

		if bs, err := fs.ReadFile(dir, fullPath); err == nil {
			return bs, nil
		}
	}

	return nil, errors.New("no such file or directory")
}
