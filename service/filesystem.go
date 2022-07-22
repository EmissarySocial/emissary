package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/config"
	"github.com/benpate/derp"
	"github.com/spf13/afero"
)

func GetFS(folder config.Folder, subFolders ...string) (afero.Fs, error) {

	subPath := strings.Join(subFolders, "/")

	switch folder.Adapter {
	case "FILE":
		return afero.NewBasePathFs(afero.NewOsFs(), folder.Location+"/"+subPath), nil

		// More to come..
		// * GitHub?
		// * S3?       https://github.com/fclairamb/afero-s3
		// * Dropbox?  https://github.com/fclairamb/afero-dropbox
		// * etc...
	}

	// Otherwise, return an error
	return nil, derp.NewInternalError("Unknown filesystem adapter: %s", folder.Adapter, folder)
}
