package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/config"
	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/afero"
)

func GetFS(folder config.Folder, subFolders ...string) afero.Fs {

	subPath := strings.Join(subFolders, "/")

	switch folder.Adapter {
	case "FILE":
		return afero.NewBasePathFs(afero.NewOsFs(), folder.Location+"/"+subPath)

		// More to come..
		// * GitHub?
		// * S3?
		// * Dropbox?
		// * etc...
	}

	panic("Unrecognized folder configuration\n" + spew.Sdump(folder))
}
