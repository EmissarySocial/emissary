package config

import "github.com/benpate/rosetta/schema"

func ReadableFolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"adapter":   schema.String{Required: true, Default: "EMBED", Enum: []string{"EMBED", "FILE", "GIT", "S3"}},
			"location":  schema.String{Required: true},
			"accessKey": schema.String{RequiredIf: "adapter is S3"},
			"secretKey": schema.String{RequiredIf: "adapter is S3"},
			"region":    schema.String{RequiredIf: "adapter is S3"},
			"token":     schema.String{RequiredIf: "adapter is S3"},
			"bucket":    schema.String{RequiredIf: "adapter is S3"},
			"path":      schema.String{RequiredIf: "adapter is S3"},
		},
	}
}

func WritableFolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"adapter":   schema.String{Required: true, Default: "FILE", Enum: []string{"FILE", "S3"}},
			"location":  schema.String{Required: true},
			"accessKey": schema.String{RequiredIf: "adapter is S3"},
			"secretKey": schema.String{RequiredIf: "adapter is S3"},
			"region":    schema.String{RequiredIf: "adapter is S3"},
			"token":     schema.String{RequiredIf: "adapter is S3"},
			"bucket":    schema.String{RequiredIf: "adapter is S3"},
			"path":      schema.String{RequiredIf: "adapter is S3"},
		},
		Wildcard: schema.String{MaxLength: 1000},
	}
}
