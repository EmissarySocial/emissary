package model

import (
	"testing"

	"github.com/benpate/rosetta/path"
	"github.com/stretchr/testify/require"
)

func TestUserAccessors(t *testing.T) {
	user := NewUser()

	require.True(t, path.SetString(&user, "userId", "123456781234567812345678"))
	require.Equal(t, "123456781234567812345678", path.GetString(&user, "userId"))

	require.True(t, path.SetString(&user, "imageId", "123456781234567812345000"))
	require.Equal(t, "123456781234567812345000", path.GetString(&user, "imageId"))

	require.True(t, path.SetString(&user, "displayName", "John Doe"))
	require.Equal(t, "John Doe", path.GetString(&user, "displayName"))

	require.True(t, path.SetString(&user, "statusMessage", "Hello World"))
	require.Equal(t, "Hello World", path.GetString(&user, "statusMessage"))

	require.True(t, path.SetString(&user, "location", "New York, NY"))
	require.Equal(t, "New York, NY", path.GetString(&user, "location"))

	require.True(t, path.SetString(&user, "links.0.inboxUrl", "https://john.doe/inbox"))
	require.Equal(t, "https://john.doe/inbox", path.GetString(&user, "links.0.inboxUrl"))

	require.True(t, path.SetString(&user, "links.0.name", "Jane Doe"))
	require.Equal(t, "Jane Doe", path.GetString(&user, "links.0.name"))

	require.True(t, path.SetString(&user, "emailAddress", "john@doe.com"))
	require.Equal(t, "john@doe.com", path.GetString(&user, "emailAddress"))

	require.True(t, path.SetString(&user, "username", "johndoe"))
	require.Equal(t, "johndoe", path.GetString(&user, "username"))

	require.True(t, path.SetString(&user, "profileUrl", "https://john.doe"))
	require.Equal(t, "https://john.doe", path.GetString(&user, "profileUrl"))

	require.True(t, path.SetBool(&user, "isOwner", true))
	require.Equal(t, true, path.GetBool(&user, "isOwner"))

	require.True(t, path.SetInt(&user, "followerCount", 123))
	require.Equal(t, 123, path.GetInt(&user, "followerCount"))

	require.True(t, path.SetInt(&user, "followingCount", 456))
	require.Equal(t, 456, path.GetInt(&user, "followingCount"))

	require.True(t, path.SetInt(&user, "blockCount", 789))
	require.Equal(t, 789, path.GetInt(&user, "blockCount"))
}
