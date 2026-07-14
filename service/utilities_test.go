package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParsePath(t *testing.T) {

	id1, _ := primitive.ObjectIDFromHex("123456789012345678901234")

	{
		urlValue, userID, objectType, objectID, err := ParseProfileURL("https://example.com/@123456789012345678901234")
		require.Nil(t, err)
		require.NotNil(t, urlValue)
		require.Equal(t, id1, userID)
		require.Empty(t, objectType)
		require.Empty(t, objectID)
	}

	{
		urlValue, userID, objectType, objectID, err := ParseProfileURL("https://example.com/@123456789012345678901234/pub")
		require.Nil(t, err)
		require.NotNil(t, urlValue)
		require.Equal(t, id1, userID)
		require.Empty(t, objectType)
		require.Empty(t, objectID)
	}
	{
		urlValue, userID, objectType, objectID, err := ParseProfileURL("https://example.com/@123456789012345678901234/pub/followers")
		require.Nil(t, err)
		require.NotNil(t, urlValue)
		require.Equal(t, id1, userID)
		require.Equal(t, "followers", objectType)
		require.Empty(t, objectID)
	}

	{
		id2, _ := primitive.ObjectIDFromHex("234567890123456789012345")

		urlValue, userID, objectType, objectID, err := ParseProfileURL("https://example.com/@123456789012345678901234/pub/followers/234567890123456789012345")
		require.Nil(t, err)
		require.NotNil(t, urlValue)
		require.Equal(t, id1, userID)
		require.Equal(t, "followers", objectType)
		require.Equal(t, id2, objectID)
	}
}

func TestParsePathErrors(t *testing.T) {
	{
		_, _, _, _, err := ParseProfileURL("not-a-url")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("https://example.com")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("https://example.com/not-a-username")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("https://example.com/@not-an-objectid")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("https://example.com/@123456789012345678901234/not-pub")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("https://example.com/@123456789012345678901234/pub/followers/not-an-objectid")
		require.NotNil(t, err)
	}

	{
		_, _, _, _, err := ParseProfileURL("https://example.com/@123456789012345678901234/pub/followers/234567890123456789012345/path-too-long")
		require.NotNil(t, err)
	}
}

func TestParseFollowersURI(t *testing.T) {

	host := "https://example.com"
	hexID := "123456789012345678901234"
	id, err := primitive.ObjectIDFromHex(hexID)
	require.Nil(t, err)

	do := func(uri string, wantType string, wantID primitive.ObjectID) {
		t.Helper()
		gotType, gotID := parseFollowersURI(host, uri)
		require.Equal(t, wantType, gotType, "actorType for %q", uri)
		require.Equal(t, wantID, gotID, "actorID for %q", uri)
	}

	// User followers collections
	do("https://example.com/@"+hexID+"/pub/followers", model.ActorTypeUser, id) // long form
	do("followers:"+hexID, model.ActorTypeUser, id)                             // shortcut scheme

	// Stream (bare hex, no "@") and SearchQuery ("@search_") followers collections — W6
	do("https://example.com/"+hexID+"/pub/followers", model.ActorTypeStream, id)
	do("https://example.com/@search_"+hexID+"/pub/followers", model.ActorTypeSearchQuery, id)

	// NOT a local followers collection → ("", NilObjectID)
	do("https://example.com/@"+hexID+"/pub/followers/", "", primitive.NilObjectID)            // trailing slash
	do("https://example.com/@"+hexID+"/invalid-other-path/", "", primitive.NilObjectID)       // wrong suffix
	do("https://example.com/@not-a-valid-userid/pub/followers", "", primitive.NilObjectID)    // non-hex user
	do("https://example.com/not-a-valid-id/pub/followers", "", primitive.NilObjectID)         // non-hex stream
	do("https://example.com/@search_not-hex/pub/followers", "", primitive.NilObjectID)        // non-hex search
	do("https://not-even-your-domain.bro/"+hexID+"/pub/followers", "", primitive.NilObjectID) // foreign host
	do("followers:not-hex", "", primitive.NilObjectID)                                        // bad shortcut token
}
