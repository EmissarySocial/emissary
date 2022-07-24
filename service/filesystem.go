package service

import (
	"io/fs"
	"os"
	"strings"

	"github.com/benpate/derp"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
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

	if strings.HasPrefix(uri, "file://") {
		uri = strings.TrimPrefix(uri, "file://")
		return os.DirFS(uri), nil
	}

	// * GitHub?
	// * S3?       https://github.com/fclairamb/afero-s3
	// * Dropbox?  https://github.com/fclairamb/afero-dropbox
	// * etc...

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

	// * GitHub?
	// * S3?       https://github.com/fclairamb/afero-s3
	// * Dropbox?  https://github.com/fclairamb/afero-dropbox
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
