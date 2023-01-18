package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestFolderSchema(t *testing.T) {

	folder := NewFolder()
	s := schema.New(FolderSchema())

	table := []tableTestItem{
		{"folderId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"label", "LABEL", nil},
		{"rank", 1.0, 1},
	}

	tableTest_Schema(t, &s, &folder, table)
}
