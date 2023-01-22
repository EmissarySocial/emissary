package config

import "github.com/benpate/rosetta/schema"

func ReadableFolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"adapter":  schema.String{Required: true, Default: "EMBED", Enum: []string{"EMBED", "FILE", "GIT", "S3"}},
			"location": schema.String{Required: true, MaxLength: 1000},
		},
	}
}

func WritableFolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"adapter":  schema.String{Required: true, Default: "FILE", Enum: []string{"FILE", "S3"}},
			"location": schema.String{Required: true, MaxLength: 1000},
		},
	}
}

func (folder Folder) GetString(name string) (string, bool) {
	switch name {

	case "adapter":
		return folder.Adapter, true

	case "location":
		return folder.Location, true
	}

	return "", false
}

func (folder *Folder) SetString(name string, value string) bool {

	switch name {
	case "adapter":
		folder.Adapter = value
		return true

	case "location":
		folder.Location = value
		return true

	}

	return false
}
