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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStreamSchema confirms that every Stream field round-trips through its JSON-Schema
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

		{"url", "https://example/document", nil},
		{"label", "DOC-LABEL", nil},
		{"summary", "DOC-SUMMARY", nil},
		{"parentTemplateId", "PARENT-TMPL", nil},
		{"context", "https://example/context", nil},
		{"icon", "doc-icon", nil}, // icon is a CSS/token name, not a URL
		{"iconUrl", "https://DOC.ICONURL.COM", nil},
		{"hashtags.0", "first-tag", nil},
		{"hashtags.1", "second-tag", nil},

		// tags is an array of objects, so each Tag property is addressed individually. The list
		// grows to fit, which is what lets a caller write tags.1.* before tags.1 exists.
		{"tags.0.type", "Hashtag", nil},
		{"tags.0.name", "first-tag", nil},
		{"tags.1.type", "Mention", nil},
		{"tags.1.name", "bob@server.social", nil},
		{"tags.1.href", "https://server.social/@bob", nil},
		// note: "isPublished" is a read-only virtual field (computed from publishDate/unpublishDate),
		// "syndication" is a delta.Slice not settable by element path, and "widgets" is a nested
		// object — none round-trip through this table helper, so they are intentionally omitted.
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

		{"data.ABC", "FIRST VALUE", nil},
		{"data.XYZ", "SECOND VALUE", nil},

		{"publishDate", 12345678, int64(12345678)},
		{"unpublishDate", 123456789, int64(123456789)},
		{"isFeatured", true, nil},
	}

	tableTest_Schema(t, &s, &stream, tests)
}

// TestStreamSchema_Aliases covers the alias properties that map onto shared fields: "name" is an
// alias for Label and "summary" maps to Summary. They share storage, so they must be
// tested in isolation (a single table would have them clobber label/summary).
func TestStreamSchema_Aliases(t *testing.T) {

	s := schema.New(StreamSchema())

	{
		stream := NewStream()
		require.Nil(t, s.Set(&stream, "name", "VIA-NAME"))
		require.Equal(t, "VIA-NAME", stream.Label) // "name" writes through to Label

		got, err := s.Get(&stream, "name")
		require.Nil(t, err)
		require.Equal(t, "VIA-NAME", got)
	}

	{
		stream := NewStream()
		require.Nil(t, s.Set(&stream, "summary", "VIA-SUMMARY"))
		require.Equal(t, "VIA-SUMMARY", stream.Summary) // "summary" writes through to Summary

		got, err := s.Get(&stream, "summary")
		require.Nil(t, err)
		require.Equal(t, "VIA-SUMMARY", got)
	}
}

// TestPermissionSchema confirms that role-to-ID permission maps round-trip through their schema
func TestPermissionSchema(t *testing.T) {

	m := mapof.NewObject[sliceof.String]()
	s := schema.New(permissionSchema())

	table := []tableTestItem{
		{"ABC.0", "12345678901234567890ABCD", nil},
		{"ABC.1", "12345678901234567890ABCD", nil},
		{"XYZ.0", "12345678901234567890ABCD", nil},
		{"XYZ.1", "12345678901234567890ABCD", nil},
	}

	tableTest_Schema(t, &s, &m, table)
}

// TestStream_IsVisibleTo confirms that DefaultAllow decides visibility for a set of permissions
func TestStream_IsVisibleTo(t *testing.T) {

	// A private Group and a signed-in User, used to build viewer permissions below.
	privateGroup := primitive.NewObjectID()
	otherGroup := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	// anonymousViewer sees the world with no signature (the /pub/children default).
	anonymousViewer := NewAnonymousPermissions()

	// memberViewer is a signed-in User who belongs to privateGroup. Mirrors how
	// Permission.ParseHTTPSignature builds a viewer: anonymous base + authenticated
	// + the User's own ID + their group memberships.
	memberViewer := append(NewAnonymousPermissions(), MagicGroupIDAuthenticated, userID, privateGroup)

	// RULE: A Stream that allows Anonymous is visible to everyone, including anonymous callers.
	{
		stream := Stream{DefaultAllow: Permissions{MagicGroupIDAnonymous}}
		require.True(t, stream.IsVisibleTo(anonymousViewer), "anonymous-allowed stream must be visible to anonymous caller")
		require.True(t, stream.IsVisibleTo(memberViewer), "anonymous-allowed stream must be visible to members")
	}

	// RULE: A Stream restricted to a Group is HIDDEN from an anonymous caller...
	{
		stream := Stream{DefaultAllow: Permissions{privateGroup}}
		require.False(t, stream.IsVisibleTo(anonymousViewer), "group-restricted stream must be hidden from anonymous caller")

		// ...but VISIBLE to a member of that Group.
		require.True(t, stream.IsVisibleTo(memberViewer), "group-restricted stream must be visible to a member of that group")
	}

	// RULE: A Stream restricted to a Group the viewer does NOT belong to is hidden.
	{
		stream := Stream{DefaultAllow: Permissions{otherGroup}}
		require.False(t, stream.IsVisibleTo(memberViewer), "stream restricted to a non-member group must be hidden")
	}

	// RULE: A Stream with an empty DefaultAllow is visible to no one (fail closed).
	{
		stream := Stream{DefaultAllow: Permissions{}}
		require.False(t, stream.IsVisibleTo(anonymousViewer), "stream with empty DefaultAllow must be hidden from anonymous caller")
		require.False(t, stream.IsVisibleTo(memberViewer), "stream with empty DefaultAllow must be hidden from members")
	}

	// RULE: An empty viewer permission set sees only... nothing (fail closed).
	{
		stream := Stream{DefaultAllow: Permissions{MagicGroupIDAnonymous}}
		require.False(t, stream.IsVisibleTo(Permissions{}), "empty viewer permissions must not match any stream")
	}
}

// TestStream_JSON confirms that a Stream survives a round-trip through JSON
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
	}, `"StartDate":"2009-11-17T20:34:58.651387237Z"`)

	test(Stream{
		EndDate: datetime.DateTime{Time: time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)},
	}, `"EndDate":"2009-11-17T20:34:58.651387237Z"`)
}
