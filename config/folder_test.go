package config

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

func TestReadableFolder(t *testing.T) {

	value := mapof.NewString()
	s := schema.New(ReadableFolderSchema())

	table := []tableTestItem{
		{"adapter", "EMBED", nil},
		{"location", "LOCATION", nil},
		{"accessKey", "ACCESS_KEY", nil},
		{"secretKey", "SECRET_KEY", nil},
		{"region", "REGION", nil},
		{"token", "TOKEN", nil},
		{"bucket", "BUCKET", nil},
		{"path", "PATH...", nil},
	}

	tableTest_Schema(t, &s, &value, table)
}

func TestWritableFolder(t *testing.T) {

	value := mapof.NewString()
	s := schema.New(WritableFolderSchema())

	table := []tableTestItem{
		{"adapter", "S3", nil},
		{"location", "LOCATION", nil},
		{"accessKey", "ACCESS_KEY", nil},
		{"secretKey", "SECRET_KEY", nil},
		{"region", "REGION", nil},
		{"token", "TOKEN", nil},
		{"bucket", "BUCKET", nil},
		{"path", "PATH...", nil},
	}

	tableTest_Schema(t, &s, &value, table)
}
