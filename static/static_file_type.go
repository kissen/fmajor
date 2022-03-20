package static

import (
	"io/fs"
	"time"
)

// Startup time. We need this for the timestamps (which are needed for etag).
var startupTime time.Time

func init() {
	startupTime = time.Now().UTC()
}

// This struct implements the fs.FileInfo interface. It mostly just forwards all
// calls to base, but ModTime() returns the process startup time instead.
type staticFileType struct {
	base fs.FileInfo
}

func (sfi *staticFileType) Name() string {
	return sfi.base.Name()
}

func (sfi *staticFileType) Size() int64 {
	return sfi.base.Size()
}

func (sfi *staticFileType) Mode() fs.FileMode {
	return sfi.base.Mode()
}

func (sfi *staticFileType) ModTime() time.Time {
	return startupTime
}

func (sfi *staticFileType) IsDir() bool {
	return sfi.base.IsDir()
}

func (sfi *staticFileType) Sys() any {
	return sfi.base.Sys()
}
