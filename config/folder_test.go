package config

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

func TestReadableFolder(t *testing.T) {

	value := mapof.Any{
		"readable": mapof.NewString(),
	}

	s := schema.New(
		schema.Object{
			Properties: schema.ElementMap{
				"readable": ReadableFolderSchema("readable"),
			},
		},
	)

	table := []tableTestItem{
		{"readable.adapter", "EMBED", nil},
		{"readable.location", "LOCATION", nil},
		{"readable.accessKey", "ACCESS_KEY", nil},
		{"readable.secretKey", "SECRET_KEY", nil},
		{"readable.region", "REGION", nil},
		{"readable.token", "TOKEN", nil},
		{"readable.bucket", "BUCKET", nil},
		{"readable.path", "PATH...", nil},
	}

	tableTest_Schema(t, &s, &value, table)
}

func TestWritableFolder(t *testing.T) {

	value := mapof.Any{
		"writable": mapof.NewString(),
	}

	s := schema.New(
		schema.Object{
			Properties: schema.ElementMap{
				"writable": WritableFolderSchema("writable"),
			},
		},
	)

	table := []tableTestItem{
		{"writable.adapter", "S3", nil},
		{"writable.location", "LOCATION", nil},
		{"writable.accessKey", "ACCESS_KEY", nil},
		{"writable.secretKey", "SECRET_KEY", nil},
		{"writable.region", "REGION", nil},
		{"writable.token", "TOKEN", nil},
		{"writable.bucket", "BUCKET", nil},
		{"writable.path", "PATH...", nil},
	}

	tableTest_Schema(t, &s, &value, table)
}
