package config

import (
	"github.com/benpate/convert"
	"github.com/benpate/derp"
)

type Folder struct {
	Adapter  string `json:"adapter"`
	Location string `json:"location"`
	Sync     bool   `json:"sync"`
}

/**************************
 * Path Interface
 **************************/

func (folder Folder) GetPath(name string) (interface{}, bool) {

	switch name {
	case "adapter":
		return folder.Adapter, true

	case "location":
		return folder.Location, true

	case "sync":
		return folder.Sync, true
	}

	return nil, false
}

func (folder *Folder) SetPath(name string, value interface{}) error {

	switch name {
	case "adapter":
		folder.Adapter = convert.String(value)
		return nil

	case "location":
		folder.Location = convert.String(value)
		return nil

	case "sync":
		folder.Sync = convert.Bool(value)
		return nil

	}

	return derp.NewInternalError("whisper.config.Folder.SetPath", "Bad path name", name, value)
}
