package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestStream_Toot_Sensitive confirms a post with a content warning is reported as sensitive
func TestStream_Toot_Sensitive(t *testing.T) {

	withWarning := NewStream()
	withWarning.Label = "Spoilers ahead"

	status := withWarning.Toot()
	require.Equal(t, "Spoilers ahead", status.SpoilerText)
	require.True(t, status.Sensitive)

	plain := NewStream().Toot()
	require.Empty(t, plain.SpoilerText)
	require.False(t, plain.Sensitive)
}

// TestUser_Toot_StaticImages confirms a local account's avatar and header also fill the static variants
func TestUser_Toot_StaticImages(t *testing.T) {

	user := NewUser()
	user.ProfileURL = "https://example.com/@me"
	user.IconID = primitive.NewObjectID()
	user.ImageID = primitive.NewObjectID()

	account := user.Toot()
	require.NotEmpty(t, account.Avatar)
	require.Equal(t, account.Avatar, account.AvatarStatic)
	require.NotEmpty(t, account.Header)
	require.Equal(t, account.Header, account.HeaderStatic)

	bare := NewUser().Toot()
	require.Empty(t, bare.AvatarStatic, "no image means no static image either")
	require.Empty(t, bare.HeaderStatic)
}

// TestPersonLink_Toot_StaticAvatar confirms a post author's avatar also fills the static variant
func TestPersonLink_Toot_StaticAvatar(t *testing.T) {

	person := PersonLink{ProfileURL: "https://example.com/@them", IconURL: "https://example.com/icon.png"}

	account := person.Toot()
	require.Equal(t, "https://example.com/icon.png", account.Avatar)
	require.Equal(t, account.Avatar, account.AvatarStatic)
}
