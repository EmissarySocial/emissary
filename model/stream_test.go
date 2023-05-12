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
		{"parentIds.0", "000000000000000000000003", nil},
		{"parentIds.2", "000000000000000000000004", nil},
		{"parentIds.3", "000000000000000000000005", nil},
		{"rank", "1234", 1234},
		{"token", "TOKEN", nil},
		{"navigationId", "000000000000000000000006", nil},
		{"templateId", "TEMPLATE", nil},
		{"socialRole", "SOCIAL-ROLE", nil},
		{"stateId", "STATE", nil},

		{"permissions.A.0", "000000000000000000000007", nil},
		{"permissions.A.1", "000000000000000000000008", nil},
		{"permissions.B.0", "000000000000000000000009", nil},
		{"permissions.B.1", "00000000000000000000000a", nil},

		{"defaultAllow.0", "00000000000000000000000b", nil},
		{"defaultAllow.1", "00000000000000000000000c", nil},

		{"url", "https://example/document", nil},
		{"label", "DOC-LABEL", nil},
		{"summary", "DOC-SUMMARY", nil},
		{"imageUrl", "DOC-IMAGEURL", nil},
		{"attributedTo.0.name", "DOC-AUTHOR-NAME", nil},
		{"attributedTo.0.profileUrl", "https://example/author", nil},

		{"inReplyTo.url", "REPLY-URL", nil},
		{"inReplyTo.label", "REPLY-LABEL", nil},
		{"inReplyTo.summary", "REPLY-SUMMARY", nil},
		{"inReplyTo.imageUrl", "REPLY-IMAGEURL", nil},
		{"inReplyTo.attributedTo.0.profileUrl", "https://example/inReplyTo", nil},
		{"inReplyTo.attributedTo.0.name", "REPLY-AUTHOR-NAME", nil},

		{"content.format", "HTML", nil},
		{"content.raw", "TEST_RAWCONTENT", nil},
		{"content.html", "TEST_HTML", nil},

		{"permissions.ABC.0", "00000000000000000000000B", nil},
		{"permissions.ABC.1", "00000000000000000000000C", nil},

		// TODO: LOW: Restore Widget test cases
		// {"widgets.ABC.0", "FIRST VALUE", nil},
		// {"widgets.ABC.1", "SECOND VALUE", nil},
		// {"widgets.XYZ.0", "THIRD VALUE", nil},
		// {"widgets.XYZ.1", "FOURTH VALUE", nil},

		{"data.ABC", "FIRST VALUE", nil},
		{"data.XYZ", "SECOND VALUE", nil},

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
