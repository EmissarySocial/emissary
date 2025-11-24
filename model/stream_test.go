package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
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
		{"rankAlt", "4321", 4321},
		{"token", "TOKEN", nil},
		{"navigationId", "000000000000000000000006", nil},
		{"templateId", "TEMPLATE", nil},
		{"socialRole", "SOCIAL-ROLE", nil},
		{"stateId", "STATE", nil},

		{"groups.A.0", "000000000000000000000007", nil},
		{"groups.A.1", "000000000000000000000008", nil},
		{"groups.B.0", "000000000000000000000009", nil},
		{"groups.B.1", "00000000000000000000000a", nil},

		{"circles.A.0", "000000000000000000000007", nil},
		{"circles.A.1", "000000000000000000000008", nil},
		{"circles.B.0", "000000000000000000000009", nil},
		{"circles.B.1", "00000000000000000000000a", nil},

		{"products.A.0", "000000000000000000000007", nil},
		{"products.A.1", "000000000000000000000008", nil},
		{"products.B.0", "000000000000000000000009", nil},
		{"products.B.1", "00000000000000000000000a", nil},

		// {"defaultAllow.0", "00000000000000000000000b", nil},
		// {"defaultAllow.1", "00000000000000000000000c", nil},

		{"url", "https://example/document", nil},
		{"label", "DOC-LABEL", nil},
		{"summary", "DOC-SUMMARY", nil},
		{"icon", "https://example/icon.png", nil},
		{"iconUrl", "https://DOC.ICONURL.COM", nil},
		{"attributedTo.name", "DOC-AUTHOR-NAME", nil},
		{"attributedTo.profileUrl", "https://example/author", nil},

		{"inReplyTo", "https://in-reply-to.com", nil},
		{"content.format", "HTML", nil},
		{"content.raw", "TEST_RAWCONTENT", nil},
		{"content.html", "TEST_HTML", nil},

		{"location.name", "The Whiskey-a-Go-Go", nil},
		{"location.formatted", "8901 Sunset Blvd, West Hollywood, CA 90069", nil},

		{"startDate.date", "2021-01-02", nil},
		{"startDate.time", "15:04", nil},
		{"startDate.datetime", "2021-01-02T15:04", nil},
		{"startDate.timezone", "UTC", nil},
		{"startDate.unix", int64(1609542240), nil},

		{"endDate.date", "2021-01-03", nil},
		{"endDate.time", "16:05", nil},
		{"endDate.datetime", "2021-01-03T16:05", nil},
		{"endDate.timezone", "UTC", nil},
		{"endDate.unix", int64(1609542240), nil},

		// TODO: LOW: Restore Widget test cases
		// {"widgets.ABC.0", "FIRST VALUE", nil},
		// {"widgets.ABC.1", "SECOND VALUE", nil},
		// {"widgets.XYZ.0", "THIRD VALUE", nil},
		// {"widgets.XYZ.1", "FOURTH VALUE", nil},

		{"data.ABC", "FIRST VALUE", nil},
		{"data.XYZ", "SECOND VALUE", nil},

		{"publishDate", 12345678, int64(12345678)},
		{"unpublishDate", 123456789, int64(123456789)},
		{"isFeatured", true, nil},
	}

	tableTest_Schema(t, &s, &stream, tests)
}

func TestPermissionSchema(t *testing.T) {

	m := mapof.NewObject[sliceof.String]()
	s := schema.New(permissionSchema())

	table := []tableTestItem{
		{"ABC.0", "FIRST VALUE", nil},
		{"ABC.1", "SECOND VALUE", nil},
		{"XYZ.0", "THIRD VALUE", nil},
		{"XYZ.1", "FOURTH VALUE", nil},
	}

	tableTest_Schema(t, &s, &m, table)
}

func TestStream_JSON(t *testing.T) {

	test := func(stream Stream, expected ...string) {
		marshaled, err := json.Marshal(stream)
		marshaledString := string(marshaled)
		require.Nil(t, err)

		for _, value := range expected {
			require.True(t, strings.Contains(marshaledString, value))
		}
	}

	test(Stream{
		StartDate: datetime.DateTime{Time: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)},
	}, `"startDate":"2009-11-17T20:34:58.651387237Z"`)

	test(Stream{
		EndDate: datetime.DateTime{Time: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)},
	}, `"endDate":"2009-11-17T20:34:58.651387237Z"`)
}
