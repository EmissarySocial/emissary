package service

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestWebFinger(t *testing.T) {

	// Discover links via WebFinger
	links, err := discoverLinks("https://mastodon.social/@benpate")

	spew.Dump(links)
	require.Nil(t, err)
	require.Equal(t, 2, len(links))

	require.Equal(t, "alternate", links[0].RelationType)
	require.Equal(t, "application/activity+json", links[0].MediaType)
	require.Equal(t, "https://mastodon.social/users/benpate", links[0].Href)

	require.Equal(t, "alternate", links[1].RelationType)
	require.Equal(t, "application/rss+xml", links[1].MediaType)
	require.Equal(t, "https://mastodon.social/@benpate", links[1].Href)
}

func TestRSS_Mastodon(t *testing.T) {

	// Discover links via WebFinger
	links, err := discoverLinks("https://mastodon.social/@benpate.rss")

	require.Nil(t, err)
	require.Equal(t, 1, len(links))
	require.Equal(t, "alternate", links[0].RelationType)
	require.Equal(t, "application/rss+xml", links[0].MediaType)
	require.Equal(t, "https://mastodon.social/@benpate.rss", links[0].Href)
}

func TestRSS_Smashing(t *testing.T) {

	// Discover links via WebFinger
	links, err := discoverLinks("https://www.smashingmagazine.com/feed/")
	require.Nil(t, err)
	require.Equal(t, 1, len(links))
	require.Equal(t, "alternate", links[0].RelationType)
	require.Equal(t, "application/xml", links[0].MediaType)
	require.Equal(t, "https://www.smashingmagazine.com/feed/", links[0].Href)
}

func TestHTMLLink_Smashing(t *testing.T) {

	// Discover links via WebFinger
	links, err := discoverLinks("https://smashingmagazine.com/")
	require.Nil(t, err)
	require.Equal(t, 1, len(links))
	require.Equal(t, "alternate", links[0].RelationType)
	require.Equal(t, "application/rss+xml", links[0].MediaType)
	require.Equal(t, "https://www.smashingmagazine.com/feed/", links[0].Href)
}

func TestHTMLLink_AppleInsider(t *testing.T) {

	// Discover links via WebFinger
	links, err := discoverLinks("https://appleinsider.com/")
	require.Nil(t, err)
	require.Equal(t, 1, len(links))
	require.Equal(t, "alternate", links[0].RelationType)
	require.Equal(t, "application/rss+xml", links[0].MediaType)
	require.Equal(t, "https://appleinsider.com/appleinsider.rss", links[0].Href)
}