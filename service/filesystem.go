package service

import (
	"io/fs"
	"net/url"
	"os"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
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

/******************************************
 * READ ONLY METHODS
 ******************************************/

// GetFS returns a READONLY Filesystem.  It works with embed:// and file:// URIs
func (filesystem *Filesystem) GetFS(folder mapof.String) (fs.FS, error) {

	switch folder["adapter"] {

	// Detect embedded file system
	case config.FolderAdapterEmbed:
		result, err := fs.Sub(filesystem.embedded, "_embed/"+folder["location"])
		return result, derp.Wrap(err, "service.Filesystem.GetFS", "Error getting filesystem", folder)

	// Detect filesystem type
	case config.FolderAdapterFile:
		return os.DirFS(folder["location"]), nil

	case config.FolderAdapterGit:
		locationURL, err := url.Parse(folder["location"])

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
func (filesystem *Filesystem) GetFSs(folders ...mapof.String) []fs.FS {

	result := make([]fs.FS, len(folders))

	for _, folder := range folders {
		if item, err := filesystem.GetFS(folder); err == nil {
			result = append(result, item)
		} else {
			derp.Report(err)
		}
	}

	return result
}

/******************************************
 * READ/WRITE METHODS
 ******************************************/

// GetAfero returns READ/WRITE a filesystem.  It works with file:// URIs
func (filesystem *Filesystem) GetAfero(folder mapof.String) (afero.Fs, error) {

	switch folder["adapter"] {

	// Detect filesystem type
	case config.FolderAdapterFile:
		return afero.NewBasePathFs(afero.NewOsFs(), folder["location"]), nil

	// Detect S3 filesystem type
	case config.FolderAdapterS3:

		// Requires:
		// accessKey
		// secretKey
		// token
		// region
		// location
		// bucket
		// path

		// Read session configuration
		config := aws.Config{
			Credentials: credentials.NewStaticCredentials(folder["accessKey"], folder["secretKey"], folder["token"]),
			Region:      pointerTo(folder["region"]),
			Endpoint:    pointerTo(folder["location"]),
		}

		// Try to make an S3 session
		session, err := session.NewSession(&config)

		if err != nil {
			return nil, derp.Wrap(err, "service.Filesystem.GetAfero", "Error creating AWS session", folder)
		}

		// Create an S3 filesystem
		result := s3.NewFs(folder["bucket"], session)

		// Force sub-directory
		return afero.NewBasePathFs(result, folder["path"]), nil
	}

	// TODO: Implement other Afero adapters to link to other cloud storage providers?
	// * HTTP? https://github.com/spf13/afero/blob/master/httpFs.go
	// * Git? https://github.com/go-git/go-git
	// * Dropbox?  https://github.com/fclairamb/afero-dropbox
	// * Google Cloud Storage? https://github.com/spf13/afero/tree/master/gcsfs
	// * SFTP? https://github.com/spf13/afero/tree/master/sftpfs
	// * Azure?
	// * etc...

	return nil, derp.NewInternalError("service.filesystem.GetAfero", "Unsupported filesystem adapter", folder)
}

// GetAferos returns multiple afero filesystems
func (filesystem *Filesystem) GetAferos(folders ...mapof.String) []afero.Fs {

	result := make([]afero.Fs, len(folders))

	for _, folder := range folders {
		if item, err := filesystem.GetAfero(folder); err == nil {
			result = append(result, item)
		} else {
			derp.Report(err)
		}
	}

	return result
}

/******************************************
 * REAL TIME WATCHING
 ******************************************/

// Watch listens to changes to this filesystem with implementation-specific adapters.  Currently only supports file:// URIs
func (filesystem *Filesystem) Watch(folder mapof.String, changed chan<- bool) error {

	if folder["adapter"] == config.FolderAdapterFile {
		return filesystem.watchOS(folder["location"], changed)
	}

	// Otherwise, this adapter doesn't support watching so just exit silently
	return nil
}

// watchOS watches a folder on the local filesystem for changes
func (filesystem *Filesystem) watchOS(uri string, changed chan<- bool) error {

	// Get all entries in the directory
	entries, err := os.ReadDir(uri)

	if err != nil {
		return derp.Wrap(err, "service.Filesystem.watchFile", "Error reading directory", uri)
	}

	// Create a new directory watcher
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		return derp.Wrap(err, "service.Filesystem.watchFile", "Error creating watcher", uri)
	}

	// Watch the top-level director
	if err := watcher.Add(uri); err != nil {
		return derp.Wrap(err, "service.Filesystem.watchFile", "Error watching directory", uri)
	}

	// Watch all sub-directories
	for _, entry := range entries {
		if entry.IsDir() {
			if err := filesystem.watchOS(uri+"/"+entry.Name(), changed); err != nil {
				derp.Report(derp.Wrap(err, "service.Filesystem.watchFile", "Error watching sub-directory", uri+"/"+entry.Name()))
			}
		}
	}

	// Background: listen for changes and pass them to the "changed" channel
	go func() {

		for {
			select {
			case <-watcher.Events:
				changed <- true

			case err := <-watcher.Errors:
				derp.Report(derp.Wrap(err, "service.Filesystem.watchFile", "Error watching directory", uri))
			}
		}
	}()

	// Success!
	return nil
}
