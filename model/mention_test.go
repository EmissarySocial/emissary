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
		{"objectId", "123456781234567812345678", nil},
		{"type", "Stream", nil},
		{"origin.type", "LIKE", nil},
		{"origin.label", "LABEL", nil},
		{"origin.url", "https://source.url", nil},
		{"origin.iconUrl", "http://entry.photo.url/", nil},
		{"author.name", "AUTHOR NAME", nil},
		{"author.emailAddress", "AUTHOR@EMAIL.COM", nil},
		{"author.profileUrl", "AUTHOR WEBSITE", nil},
		{"author.iconUrl", "AUTHOR PHOTO", nil},
	}

	tableTest_Schema(t, &s, &mention, table)
}
