//go:build localonly

package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestWebFinger(t *testing.T) {

	// Discover links via WebFinger
	links := discoverLinks("https://mastodon.social/@benpate")

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
	links := discoverLinks("https://mastodon.social/@benpate.rss")

	require.Equal(t, 1, len(links))
	require.Equal(t, "self", links[0].RelationType)
	require.Equal(t, "application/rss+xml", links[0].MediaType)
	require.Equal(t, "https://mastodon.social/@benpate.rss", links[0].Href)
}

func TestRSS_Smashing(t *testing.T) {

	// Discover links via WebFinger
	links := discoverLinks("https://www.smashingmagazine.com/feed/")

	require.Equal(t, 1, len(links))
	require.Equal(t, "self", links[0].RelationType)
	require.Equal(t, "application/xml", links[0].MediaType)
	require.Equal(t, "https://www.smashingmagazine.com/feed/", links[0].Href)
}

func TestHTMLLink_Smashing(t *testing.T) {

	// Discover links via WebFinger
	links := discoverLinks("https://smashingmagazine.com/")

	require.Equal(t, 1, len(links))
	require.Equal(t, "alternate", links[0].RelationType)
	require.Equal(t, "application/rss+xml", links[0].MediaType)
	require.Equal(t, "https://www.smashingmagazine.com/feed/", links[0].Href)
}

func TestHTMLLink_AppleInsider(t *testing.T) {

	// Discover links via WebFinger
	links := discoverLinks("https://appleinsider.com/")

	require.Equal(t, 1, len(links))
	require.Equal(t, "alternate", links[0].RelationType)
	require.Equal(t, "application/rss+xml", links[0].MediaType)
	require.Equal(t, "https://appleinsider.com/appleinsider.rss", links[0].Href)
}

func TestWebSub_WebSubRocks_Header(t *testing.T) {

	links := discoverLinks("https://websub.rocks/blog/100/z6OK77IFLM2fWS5mJiKq")

	require.Equal(t, 2, len(links))

	require.Equal(t, "hub", links[0].RelationType)
	require.Equal(t, model.MagicMimeTypeWebSub, links[0].MediaType)
	require.Equal(t, "https://websub.rocks/blog/100/z6OK77IFLM2fWS5mJiKq/hub", links[0].Href)

	require.Equal(t, "self", links[1].RelationType)
	require.Equal(t, "", links[1].MediaType)
	require.Equal(t, "https://websub.rocks/blog/100/z6OK77IFLM2fWS5mJiKq", links[1].Href)
}

func TestWebSub_WebSubRocks_Misc(t *testing.T) {

	links := discoverLinks("https://websub.rocks/blog/204/V3ISV7nK4tBgZ9Jf2TSi")

	spew.Dump(links)

	require.Equal(t, 2, len(links))
}

func TestWebSub_WebSubRocks_RSS(t *testing.T) {

	links := discoverLinks("https://websub.rocks/blog/103/jHUNH8w2MbV1CjuwQ5Nx")

	spew.Dump(links)

	require.Equal(t, 2, len(links))

	require.Equal(t, "hub", links[0].RelationType)
	require.Equal(t, model.MagicMimeTypeWebSub, links[0].MediaType)
	require.Equal(t, "https://websub.rocks/blog/103/jHUNH8w2MbV1CjuwQ5Nx/hub", links[0].Href)

	require.Equal(t, "self", links[1].RelationType)
	require.Equal(t, "", links[1].MediaType)
	require.Equal(t, "https://websub.rocks/blog/103/jHUNH8w2MbV1CjuwQ5Nx", links[1].Href)
}
