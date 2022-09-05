package service

import (
	"io/fs"
	"os"
	"strings"

	"github.com/EmissarySocial/emissary/tools/s3uri"
	"github.com/benpate/derp"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"

	s3 "github.com/fclairamb/afero-s3"
)

// Filesystem is a service that multiplexes between different filesystems.  Currently works with embedded filesystems and file:// URIs
type Filesystem struct {
	system fs.FS
}

// NewFilesytem returns a fully initialized Filesystem service
func NewFilesystem(system fs.FS) Filesystem {

	return Filesystem{
		system: system,
	}
}

/*******************************************
 * READ ONLY METHODS
 *******************************************/

// GetFS returns a READONLY Filesystem.  It works with embed:// and file:// URIs
func (filesystem *Filesystem) GetFS(uri string) (fs.FS, error) {

	// Detect embedded file system
	if strings.HasPrefix(uri, "embed://") {
		uri = strings.TrimPrefix(uri, "embed://")
		result, err := fs.Sub(filesystem.system, "/_embed/"+uri)
		return result, derp.Wrap(err, "service.Filesystem.GetFS", "Error getting filesystem", uri)
	}

	// Detect filesystem type
	if strings.HasPrefix(uri, "file://") {
		uri = strings.TrimPrefix(uri, "file://")
		return os.DirFS(uri), nil
	}

	// Pass through to afero (create a read-only filesystem)
	if result, err := filesystem.GetAfero(uri); err == nil {
		return afero.NewIOFS(result), nil
	}

	// Otherwise, fail.  Unrecognized filesystem type
	return nil, derp.NewInternalError("service.filesystem.GetFS", "Unsupported filesystem adapter", uri)
}

// GetFSs returns multiple fs.FS filesystems
func (filesystem *Filesystem) GetFSs(urls ...string) ([]fs.FS, error) {

	result := make([]fs.FS, len(urls))
	var errAcc error

	for i, url := range urls {
		item, err := filesystem.GetFS(url)
		result[i] = item
		errAcc = derp.Append(errAcc, err)
	}

	return result, errAcc
}

/*******************************************
 * READ/WRITE METHODS
 *******************************************/

// GetAfero returns READ/WRITE a filesystem.  It works with file:// URIs
func (filesystem *Filesystem) GetAfero(uri string) (afero.Fs, error) {

	// Detect filesystem type
	if strings.HasPrefix(uri, "file://") {
		trimmed := strings.TrimPrefix(uri, "file://")
		return afero.NewBasePathFs(afero.NewOsFs(), trimmed), nil
	}

	// Detect S3 filesystem type
	if uri, err := s3uri.ParseString(uri); err == nil {

		// Read session configuration
		config := aws.Config{Region: uri.Region}

		if uri.HasCredentials() {
			config.Credentials = credentials.NewStaticCredentials(uri.GetCredentials())
		}

		// Try to make an S3 session
		session, err := session.NewSession(&config)

		if err != nil {
			return nil, derp.Wrap(err, "service.Filesystem.GetAfero", "Error creating AWS session", uri)
		}

		// Create an S3 filesystem
		return s3.NewFs(*uri.Bucket, session), nil
	}

	// * HTTP? https://github.com/spf13/afero/blob/master/httpFs.go
	// * Git? https://github.com/go-git/go-git
	// * Dropbox?  https://github.com/fclairamb/afero-dropbox
	// * Google Cloud Storage? https://github.com/spf13/afero/tree/master/gcsfs
	// * SFTP? https://github.com/spf13/afero/tree/master/sftpfs
	// * Azure?
	// * etc...

	return nil, derp.NewInternalError("service.filesystem.GetAfero", "Unsupported filesystem adapter", uri)
}

// GetAferos returns multiple afero filesystems
func (filesystem *Filesystem) GetAferos(uris ...string) ([]afero.Fs, error) {

	result := make([]afero.Fs, len(uris))
	var errAcc error

	for i, url := range uris {
		item, err := filesystem.GetAfero(url)
		result[i] = item
		errAcc = derp.Append(errAcc, err)
	}

	return result, errAcc
}

/*******************************************
 * REAL TIME WATCHING
 *******************************************/

func (filesystem *Filesystem) Watch(uri string, changed chan<- bool, closed <-chan bool) error {

	if strings.HasPrefix(uri, "file://") {
		uri = strings.TrimPrefix(uri, "file://")
		return filesystem.watchOS(uri, changed, closed)
	}

	return derp.NewInternalError("service.Filesystem.Watch", "Unsupported filesystem adapter", uri)
}

func (filesystem *Filesystem) watchOS(uri string, changed chan<- bool, closed <-chan bool) error {

	// Create a new directory watcher
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return derp.Wrap(err, "service.Filesystem.watchFile", "Error creating watcher", uri)
	}

	entries, err := os.ReadDir(uri)

	if err != nil {
		return derp.Wrap(err, "service.Filesystem.watchFile", "Error reading directory", uri)
	}

	for _, entry := range entries {
		watcher.Add(uri + "/" + entry.Name())
	}

	// Watch for events
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				changed <- true
			}
		case err := <-watcher.Errors:
			return derp.Wrap(err, "service.Filesystem.watchFile", "Error watching directory", uri)

		case <-closed:
			close(changed)
			return nil
		}
	}
}
