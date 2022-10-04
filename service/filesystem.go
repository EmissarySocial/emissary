package service

import (
	"io/fs"
	"net/url"
	"os"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/tools/s3uri"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/hairyhenderson/go-fsimpl/gitfs"

	s3 "github.com/fclairamb/afero-s3"
)

// Filesystem is a service that multiplexes between different filesystems.  Currently works with embedded filesystems and file:// URIs
type Filesystem struct {
	embedded fs.FS
}

// NewFilesytem returns a fully initialized Filesystem service
func NewFilesystem(embedded fs.FS) Filesystem {

	return Filesystem{
		embedded: embedded,
	}
}

/*******************************************
 * READ ONLY METHODS
 *******************************************/

// GetFS returns a READONLY Filesystem.  It works with embed:// and file:// URIs
func (filesystem *Filesystem) GetFS(folder config.Folder) (fs.FS, error) {

	switch folder.Adapter {

	// Detect embedded file system
	case config.FolderAdapterEmbed:
		result, err := fs.Sub(filesystem.embedded, "_embed/"+folder.Location)
		return result, derp.Wrap(err, "service.Filesystem.GetFS", "Error getting filesystem", folder)

	// Detect filesystem type
	case config.FolderAdapterFile:
		return os.DirFS(folder.Location), nil

	case config.FolderAdapterGit:
		locationURL, err := url.Parse(folder.Location)

		if err != nil {
			return nil, derp.Wrap(err, "service.Filesystem.GetFS", "Error parsing Git URL", folder)
		}

		return gitfs.New(locationURL)
	}

	// Otherwise, pass through to afero (create a read-only filesystem)
	if result, err := filesystem.GetAfero(folder); err == nil {
		return afero.NewIOFS(result), nil
	}

	// Otherwise, fail.  Unrecognized filesystem type
	return nil, derp.NewInternalError("service.filesystem.GetFS", "Unsupported filesystem adapter", folder)
}

// GetFSs returns multiple fs.FS filesystems
func (filesystem *Filesystem) GetFSs(folders ...config.Folder) ([]fs.FS, error) {

	result := make([]fs.FS, len(folders))
	var errAcc error

	for i, folder := range folders {
		item, err := filesystem.GetFS(folder)
		result[i] = item
		errAcc = derp.Append(errAcc, err)
	}

	return result, errAcc
}

/*******************************************
 * READ/WRITE METHODS
 *******************************************/

// GetAfero returns READ/WRITE a filesystem.  It works with file:// URIs
func (filesystem *Filesystem) GetAfero(folder config.Folder) (afero.Fs, error) {

	switch folder.Adapter {

	// Detect filesystem type
	case config.FolderAdapterFile:
		return afero.NewBasePathFs(afero.NewOsFs(), folder.Location), nil

	// Detect S3 filesystem type
	case config.FolderAdapterS3:
		uri, err := s3uri.ParseString(folder.Location)

		if err != nil {
			return nil, derp.Wrap(err, "service.Filesystem.GetAfero", "Error parsing S3 URI", uri)
		}

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

		// * HTTP? https://github.com/spf13/afero/blob/master/httpFs.go
		// * Git? https://github.com/go-git/go-git
		// * Dropbox?  https://github.com/fclairamb/afero-dropbox
		// * Google Cloud Storage? https://github.com/spf13/afero/tree/master/gcsfs
		// * SFTP? https://github.com/spf13/afero/tree/master/sftpfs
		// * Azure?
		// * etc...
	}

	return nil, derp.NewInternalError("service.filesystem.GetAfero", "Unsupported filesystem adapter", folder)
}

// GetAferos returns multiple afero filesystems
func (filesystem *Filesystem) GetAferos(folders ...config.Folder) ([]afero.Fs, error) {

	result := make([]afero.Fs, len(folders))
	var errAcc error

	for i, folder := range folders {
		item, err := filesystem.GetAfero(folder)
		result[i] = item
		errAcc = derp.Append(errAcc, err)
	}

	return result, errAcc
}

/*******************************************
 * REAL TIME WATCHING
 *******************************************/

func (filesystem *Filesystem) Watch(folder config.Folder, changed chan<- bool, closed <-chan bool) error {

	if folder.Adapter == config.FolderAdapterFile {
		return filesystem.watchOS(folder.Location, changed, closed)
	}

	// Otherwise, this adapter doesn't support watching so just exit silently
	return nil
}

func (filesystem *Filesystem) watchOS(uri string, changed chan<- bool, closed <-chan bool) error {

	spew.Dump("Watching", uri)

	// Create a new directory watcher
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return derp.Wrap(err, "service.Filesystem.watchFile", "Error creating watcher", uri)
	}

	entries, err := os.ReadDir(uri)

	if err != nil {
		return derp.Wrap(err, "service.Filesystem.watchFile", "Error reading directory", uri)
	}

	watcher.Add(uri)

	for _, entry := range entries {
		if entry.IsDir() {
			watcher.Add(uri + "/" + entry.Name())
		}
	}

	// Watch for events
	for {
		select {
		case <-watcher.Events:
			changed <- true

		case err := <-watcher.Errors:
			return derp.Wrap(err, "service.Filesystem.watchFile", "Error watching directory", uri)

		case <-closed:
			close(changed)
			return nil
		}
	}
}
