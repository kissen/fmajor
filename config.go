package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Configuration of a fmajor instance.
type Config struct {
	// The address to listen on for HTTP connections. Probably
	// something like "localhost:9090".
	ListenAddress string

	// The directory where to put files and metadata.
	//
	// The process running fmajor will need rw permissions
	// on this directory.
	UploadsDirectory string

	// Maximum file size in bytes. Stored as signed integer
	// because the http API where we use MaxFileSize requires
	// a signed integer.
	MaxFileSize int64

	// Set of bcrypt password hashes. For example, you can create
	// these hashes by running:
	//
	//   htpasswd -n -B -C 12 "" | tr -d ':\n'
	//
	// The generated hash should look something like this:
	//
	//   $2y$12$BkkH3A/W67qKQ7vwCxwcPOf4XllhwNWxTV5Pl4Zb1aLd1bd4Ga5m2
	//
	PassHashes []string
}

// Global instance of the configuration. Use GetConfig to access
// this variable.
var config *Config
var configCreator sync.Once

// Return the singleton instance of the config.
func GetConfig() *Config {
	configCreator.Do(loadConfig)
	return config
}

// Return any errors in the configuration if we can find them.
// This is only a basic test for empty strings and the like.
func (c *Config) Error() error {
	if c.ListenAddress == "" {
		return errors.New("empty ListenAddress")
	}

	if c.UploadsDirectory == "" {
		return errors.New("empty UploadsDirecotry")
	}

	if c.MaxFileSize <= 0 {
		return fmt.Errorf("bad MaxFileSize=%v", c.MaxFileSize)
	}

	if len(c.PassHashes) == 0 {
		return errors.New("empty PassHashes")
	}

	return nil
}

// Populate the "config" global variable. If it fails, we can't continue,
// in that case we stop the program.
func loadConfig() {
	// try to find a config file that works; if it fails we don't
	// report an error right away as a later file might be available

	var errs []error

	for _, path := range configPaths() {
		if c, err := loadConfigFrom(path); err != nil {
			errs = append(errs, err)
		} else {
			config = c
			break
		}
	}

	// if we failed to find any usable config file, barf out
	// all collected errors and quit

	if config == nil {
		for _, err := range errs {
			log.Println(err)
		}

		log.Fatal("missing configuration file, maybe supply one with the -c flag")
	}
}

// Return a collection of filepaths where we may find a configuration
// file to parse.
func configPaths() []string {
	// first check whether the user supplied the -c flag; if they did,
	// only consider that file

	var custom string
	flag.StringVar(&custom, "c", "/dev/null", "path to configuration file")
	flag.Parse()
	if custom != "/dev/null" {
		return []string{
			custom,
		}
	}

	// if no particular file was supplied, try out the default files

	var paths []string
	filename := "fmajor.conf"

	if udir, err := os.UserConfigDir(); err == nil {
		p := filepath.Join(udir, filename)
		paths = append(paths, p)
	}

	sysdir := "/etc"
	p := filepath.Join(sysdir, filename)
	paths = append(paths, p)

	return paths
}

// Try to open and parse configuration file at filename.
func loadConfigFrom(filename string) (*Config, error) {
	var c Config

	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "could not load config file")
	}

	if err := toml.Unmarshal(bs, &c); err != nil {
		return nil, errors.Wrapf(err, `filename="%v" not a valid config file`, filename)
	}

	if err := c.Error(); err != nil {
		return nil, errors.Wrapf(err, `filename="%v" not a valid config file`, filename)
	}

	return &c, nil
}
