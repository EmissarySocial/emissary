package config

import (
	"github.com/benpate/derp"
	"github.com/spf13/afero"
)

// FolderAdapterOS represents files that are stored directly in the os filesystem.
const FolderAdapterOS = "FILE"

// Other adapter types to come

type Folder struct {
	Adapter  string `path:"adapter"  json:"adapter"`
	Location string `path:"location" json:"location"`
	Sync     bool   `path:"sync"     json:"sync"`
}

// GetFilesystem uses the folder's configuration to create an afero.Fs
// object that can read/write files.
func (folder Folder) GetFilesystem() (afero.Fs, error) {

	var base afero.Fs

	// Find the right base filesystem adapter
	switch folder.Adapter {

	case FolderAdapterOS:
		base = afero.NewOsFs()

	default:
		return nil, derp.NewInternalError("config.Folder.GetFilesystem", "Unrecognized Adapter", folder)

	}

	// Try to make a new subfolder at the chosen path (returns nil if already exists)
	if err := base.MkdirAll(folder.Location, 0777); err != nil {
		return nil, derp.Wrap(err, "config.Folder.GetFilesystem", "Error creating subdirectory", folder)
	}

	// Return a filesystem pointing to the new subfolder.
	return afero.NewBasePathFs(base, folder.Location), nil
}
