package main

// Configuration of a fmajor instance.
type Config struct {
	// The address to listen for HTTP connections. Probably
	// something like "http://localhost:1234".
	ListenAddress string

	// The directory where to put files and metadata.
	//
	// The process running fmajor will need rw permissions
	// on this directory.
	UploadsDirectory string

	// Maximum file size in bytes.
	MaxFileSize int64
}

func GetConfig() Config {
	return Config{
		ListenAddress:    "localhost:8080",
		UploadsDirectory: "/var/tmp",
		MaxFileSize:      1024 * 1024 * 1024 * 128, // 128 MiB
	}
}
