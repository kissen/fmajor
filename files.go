package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/TwiN/go-away"
	"github.com/dchest/uniuri"
	"github.com/disintegration/imaging"
	"github.com/docker/go-units"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/kissen/stringset"
	"github.com/pkg/errors"
)

// Maximum size an image may be for us to compute a thumbnail.  Unfortunately we
// have to load all of the image into memory before we can compute the thumbnail
// so this limitation is necessary.
const MAX_THUMBNAIL_SOURCE_SIZE = 32 * units.MiB

// Maximum width and height of a generated thumbnail in pixels.
const MAX_THUMBNAIL_DIM = 128

// Minimum width and height of a generated thumbnail in pixels. If the generated
// thumbnail would be smaller (as to maintain aspect ratio), no thumbnail is
// generated.
const MIN_THUMBNAIL_DIM = 16

// Contains all mime types for which File.Inline should return
// true, that is those mime types that should be shown inline (in
// the web browser). Initialized in init().
var inlineMimeTypes stringset.StringSet

// Contains image mime types.
var imageMimeTypes stringset.StringSet

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

	imageMimeTypes = stringset.NewWith(
		"image/bmp", "image/gif", "image/jpeg", "image/png",
		"image/webp",
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

	// Absolute filepath to the thumbnail. May be nil.
	ThumbnailPath *string

	// Thumbnail size in bytes. May be nil.
	ThumbnailSize *int64

	// Shortened Id.
	ShortId *string
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

// Return whether the underlying File is an image. This is used to determine
// whether a thumbnail should be displayed.
func (f *File) IsImage() bool {
	return imageMimeTypes.Contains(f.ContentType)
}

func (f *File) HasThumbnail() bool {
	return f.ThumbnailPath != nil
}

func (f *File) HasShortUrl() bool {
	return f.ShortId != nil
}

func (f *File) ShortUrl() string {
	host := GetConfig().HostName
	id := f.ShortId

	return path.Join(host, "f", *id)
}

func (f *File) LocalPath() string {
	parent := GetConfig().UploadsDirectory
	return filepath.Join(parent, f.Id, "storage.bin")
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
		if isSymlink(fi) {
			continue
		}

		if !isDir(fi) {
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
func CreateFile(src io.Reader, filename string, createShortId bool) (*File, error) {
	// figure out meta data

	id := uuid.New().String()

	storageDir := GetConfig().UploadsDirectory
	baseDir := filepath.Join(storageDir, id)
	metaPath := filepath.Join(baseDir, "meta.json")
	storagePath := filepath.Join(baseDir, "storage.bin")
	thumbnailPath := filepath.Join(baseDir, "thumbnail.jpg")

	// create the directory where all related files are going to be stored

	if err := os.Mkdir(baseDir, 0700); err != nil {
		return nil, errors.Wrapf(err, `cannot create directory for filename="%v"`, filename)
	}

	// copy in the actual file

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

	// create the meta object

	meta := File{
		Id:            id,
		Name:          filename,
		Size:          nbytes,
		UploadedOnUTC: time.Now().UTC(),
		ContentType:   mime.TypeByExtension(path.Ext(filename)),
	}

	if meta.ContentType == "" {
		meta.ContentType = "application/octet-stream"
	}

	// create thumbnail if necessary

	if meta.IsImage() {
		meta.ThumbnailPath = &thumbnailPath

		if err = createThumbnailFor(&meta, thumbnailPath); err != nil {
			DeleteFileAsync(id)
			return nil, errors.Wrapf(err, "could not create thumbnail")
		}
	}

	// create short link if requested

	if createShortId {
		if err = createShortIdFor(&meta); err != nil {
			DeleteFile(id)
			return nil, errors.Wrapf(err, "could not create short id")
		}
	}

	// write out meta object as json

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
	metaErr := os.Remove(metaPath)

	storagePath := filepath.Join(baseDir, "storage.bin")
	storageErr := os.Remove(storagePath)

	thumbnailPath := filepath.Join(baseDir, "thumbnail.jpg")
	os.Remove(thumbnailPath) // thumbnail might not exist

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
		os.Remove(metaPath)

		storagePath := filepath.Join(baseDir, "storage.bin")
		os.Remove(storagePath)

		thumbnailPath := filepath.Join(baseDir, "thumbnail.jpg")
		os.Remove(thumbnailPath)

		if err := os.Remove(baseDir); err != nil {
			log.Printf(`could not clean up id="%v"`, id)
		}
	}()
}

func createThumbnailFor(meta *File, filepath string) error {
	var (
		err    error
		fi     os.FileInfo
		fp     *os.File
		source image.Image
	)

	// open image file

	if fp, err = os.Open(meta.LocalPath()); err != nil {
		return errors.Wrap(err, "could not open file")
	}

	defer fp.Close()

	// parse image

	reader := &io.LimitedReader{R: fp, N: MAX_THUMBNAIL_SOURCE_SIZE}

	if source, _, err = image.Decode(reader); err != nil {
		return errors.Wrap(err, "could not decode image")
	}

	// check dimensions

	sourceWidth := source.Bounds().Max.X
	sourceHeight := source.Bounds().Max.Y

	if sourceWidth <= 0 || sourceHeight <= 0 {
		return fmt.Errorf("bad dimensions %v x %v", sourceWidth, sourceHeight)
	}

	// compute thumbnail dimensions

	scale := 1.0

	if sourceWidth > sourceHeight {
		scale = MAX_THUMBNAIL_DIM / float64(sourceWidth)
	} else {
		scale = MAX_THUMBNAIL_DIM / float64(sourceHeight)
	}

	thumbWidth := int(float64(sourceWidth) * scale)
	thumbHeight := int(float64(sourceHeight) * scale)

	if thumbWidth < MIN_THUMBNAIL_DIM || thumbHeight < MIN_THUMBNAIL_DIM {
		return fmt.Errorf("bad thumbnail dimensions %v x %v", thumbWidth, thumbHeight)
	}

	// compute and save the thumbnail

	thumbnail := imaging.Thumbnail(source, thumbWidth, thumbHeight, imaging.Lanczos)

	if err = imaging.Save(thumbnail, filepath); err != nil {
		return errors.Wrapf(err, `could not save thumbnail to "%v"`, filepath)
	}

	// update the thumbnail size

	if fi, err = os.Stat(*meta.ThumbnailPath); err != nil {
		return errors.Wrapf(err, `could not stat thumbnail for id="%v"`, meta.Id)
	}

	thumbnailSize := fi.Size()
	meta.ThumbnailSize = &thumbnailSize

	// success

	return nil
}

func createShortIdFor(meta *File) error {
	// First figure out to where we want to make a symlink.

	storageDir := GetConfig().UploadsDirectory
	targetDir := filepath.Join(storageDir, meta.Id)

	// Now keep trying to get an acceptable short id.

	lens := []int{3, 3, 4, 4, 4, 5, 5, 5, 5, 6, 7, 8, 9}

	for _, choiceLen := range lens {
		linkName := createRandomString(choiceLen)
		linkDir := filepath.Join(storageDir, linkName)

		if err := os.Symlink(targetDir, linkDir); err == nil {
			meta.ShortId = &linkName

			return nil
		}
	}

	// We found no unique id :'(

	return errors.New("could not generate unique short id")
}

func createRandomString(len int) string {
	choice := uniuri.NewLen(len)

	for goaway.IsProfane(choice) {
		choice = uniuri.NewLen(len)
	}

	return choice
}

func isSymlink(fi os.FileInfo) bool {
	return (fi.Mode() & fs.ModeSymlink) != 0
}

func isDir(fi os.FileInfo) bool {
	return fi.IsDir()
}
