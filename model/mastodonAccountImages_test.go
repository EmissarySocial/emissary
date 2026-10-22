package model

import (
	"testing"
	"time"

	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// testHost stands in for this server's host in the default image URLs
const testHost = "https://example.com"

// requireAccountImages fails unless every avatar and header field is a URL on testHost's defaults or a real image
func requireAccountImages(t *testing.T, name string, account object.Account) {

	t.Helper()

	require.NotEmpty(t, account.Avatar, name+": avatar")
	require.NotEmpty(t, account.AvatarStatic, name+": avatar_static")
	require.NotEmpty(t, account.Header, name+": header")
	require.NotEmpty(t, account.HeaderStatic, name+": header_static")
}

// TestMastodonAccounts_NeverEmptyImages builds each kind of Mastodon account with no images and requires defaults
func TestMastodonAccounts_NeverEmptyImages(t *testing.T) {

	// Every builder that emits an Account, directly or embedded in a Status or Notification
	accounts := map[string]object.Account{
		"User":               NewUser().Toot(testHost),
		"PersonLink":         PersonLink{ProfileURL: "https://remote.example/@someone"}.Toot(testHost),
		"RemoteActorAccount": RemoteActorAccount("https://remote.example/@someone", "", "", time.Time{}, testHost),
		"Stream":             NewStream().Toot(testHost).Account,
		"NewsItem":           NewNewsItem().Toot(testHost).Account,
		"Notification":       NewNotification().Toot(testHost).Account,
		"Response":           NewResponse().Toot(testHost).Account,
	}

	for name, account := range accounts {
		requireAccountImages(t, name, account)
		require.Equal(t, testHost+MastodonDefaultAvatarPath, account.Avatar, name)
		require.Equal(t, testHost+MastodonDefaultHeaderPath, account.Header, name)
	}
}

// TestWithMastodonDefaultImages_KeepsRealImages confirms an account's own images are never replaced
func TestWithMastodonDefaultImages_KeepsRealImages(t *testing.T) {

	account := WithMastodonDefaultImages(object.Account{
		Avatar:       "https://remote.example/avatar.png",
		AvatarStatic: "https://remote.example/avatar-static.png",
		Header:       "https://remote.example/header.png",
	}, testHost)

	require.Equal(t, "https://remote.example/avatar.png", account.Avatar)
	require.Equal(t, "https://remote.example/avatar-static.png", account.AvatarStatic)
	require.Equal(t, "https://remote.example/header.png", account.Header)

	// A missing static copy falls back to the real image, not the default
	require.Equal(t, "https://remote.example/header.png", account.HeaderStatic)
}
