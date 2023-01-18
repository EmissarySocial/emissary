package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestMention(t *testing.T) {

	mention := NewMention()

	s := schema.New(MentionSchema())

	table := []tableTestItem{
		{"mentionId", "123412341234123412341234", nil},
		{"streamId", "123456781234567812345678", nil},
		{"originUrl", "https://source.url", nil},
		{"authorName", "AUTHOR NAME", nil},
		{"authorEmail", "AUTHOR EMAIL", nil},
		{"authorWebsiteUrl", "AUTHOR WEBSITE", nil},
		{"authorPhotoUrl", "AUTHOR PHOTO", nil},
		{"authorStatus", "AUTHOR STATUS", nil},
		{"entryName", "ENTRY NAME", nil},
		{"entryPhotoUrl", "http://entry.photo.url/", nil},
	}

	tableTest_Schema(t, &s, &mention, table)
}
