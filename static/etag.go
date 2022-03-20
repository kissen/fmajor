package static

import (
	"fmt"
	"io/fs"
)

// Compute a reasonable ETag header value for fi.
func EtagFor(fi fs.FileInfo) string {
	name := fi.Name()
	timestamp := fi.ModTime().Unix()
	size := fi.Size()

	return fmt.Sprintf("%v-%v-%v", name, timestamp, size)
}
