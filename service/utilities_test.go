package service

import (
	"testing"

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

	{
		userID := parseFollowersURI(host, "https://example.com/@123456789012345678901234/pub/followers")
		expectedID, _ := primitive.ObjectIDFromHex("123456789012345678901234")
		require.Equal(t, expectedID, userID)
	}

	{
		userID := parseFollowersURI(host, "https://example.com/@123456789012345678901234/pub/followers/")
		require.Zero(t, userID)
	}

	{
		userID := parseFollowersURI(host, "https://example.com/@123456789012345678901234/invalid-other-path/")
		require.Zero(t, userID)
	}

	{
		userID := parseFollowersURI(host, "https://example.com/@not-a-valid-userid/pub/followers/")
		require.Zero(t, userID)
	}

	{
		userID := parseFollowersURI(host, "https://not-even-your-domain.bro")
		require.Zero(t, userID)
	}

}
