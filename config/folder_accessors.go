package config

import "github.com/benpate/rosetta/schema"

func ReadableFolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"adapter": schema.String{Required: true, Default: "EMBED", Enum: []string{"EMBED", "FILE", "GIT", "S3"}},
		},
		Wildcard: schema.String{MaxLength: 1000},
	}
}

func WritableFolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"adapter": schema.String{Required: true, Default: "FILE", Enum: []string{"FILE", "S3"}},
		},
		Wildcard: schema.String{MaxLength: 1000},
	}
}
