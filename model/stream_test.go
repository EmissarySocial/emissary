package model

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
)

func TestStreamSchema(t *testing.T) {

	s := schema.New(StreamSchema())
	stream := NewStream()

	tests := []tableTestItem{
		{"streamId", "000000000000000000000001", nil},
		{"parentId", "000000000000000000000002", nil},
		{"token", "TOKEN", nil},
		{"navigationId", "000000000000000000000003", nil},
		{"templateId", "TEMPLATE", nil},
		{"stateId", "STATE", nil},

		{"document.internalId", "000000000000000000000004", nil},
		{"document.url", "https://example/document", nil},
		{"document.label", "DOC-LABEL", nil},
		{"document.summary", "DOC-SUMMARY", nil},
		{"document.imageUrl", "DOC-IMAGEURL", nil},
		{"document.author.name", "DOC-AUTHOR-NAME", nil},

		{"replyTo.url", "https://example/replyTo", nil},
		{"replyTo.label", "REPLY-LABEL", nil},
		{"replyTo.summary", "REPLY-SUMMARY", nil},
		{"replyTo.imageUrl", "REPLY-IMAGEURL", nil},
		{"replyTo.author.name", "REPLY-AUTHOR-NAME", nil},

		{"content.format", "HTML", nil},
		{"content.raw", "TEST_RAWCONTENT", nil},
		{"content.html", "TEST_HTML", nil},

		{"rank", "1234", 1234},
		{"publishDate", 12345678, int64(12345678)},
		{"unpublishDate", 123456789, int64(123456789)},
	}

	tableTest_Schema(t, &s, &stream, tests)
}

func TestPermissionSchema(t *testing.T) {

	m := mapof.NewObject[sliceof.String]()
	s := schema.New(PermissionSchema())

	table := []tableTestItem{
		{"ABC.0", "FIRST VALUE", nil},
		{"ABC.1", "SECOND VALUE", nil},
		{"XYZ.0", "THIRD VALUE", nil},
		{"XYZ.1", "FOURTH VALUE", nil},
	}

	tableTest_Schema(t, &s, &m, table)
}
