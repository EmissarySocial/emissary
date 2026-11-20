package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestLinkPreview_Card confirms only a READY preview becomes a card, and fills its required fields
func TestLinkPreview_Card(t *testing.T) {

	failed := LinkPreview{URL: "https://news.example/story", Status: LinkPreviewStatusFailed}
	require.Nil(t, failed.Card())

	ready := LinkPreview{URL: "https://news.example/story?ref=feed", Status: LinkPreviewStatusReady, Title: "A Good Story", AuthorName: "Pat"}
	card := ready.Card()

	require.NotNil(t, card)
	require.Equal(t, "https://news.example/story?ref=feed", card.URL, "with no canonical address the link itself is used")
	require.Equal(t, "link", card.Type)
	require.Len(t, card.Authors, 1)

	// Clients decode authors as an array, so it is never null
	require.NotNil(t, LinkPreview{Status: LinkPreviewStatusReady, Title: "x"}.Card().Authors)
}

// TestLinkPreview_IsFresh confirms a record is usable until it expires
func TestLinkPreview_IsFresh(t *testing.T) {

	now := time.Now()
	require.True(t, LinkPreview{ExpiresAt: now.Add(time.Minute)}.IsFresh(now))
	require.False(t, LinkPreview{ExpiresAt: now.Add(-time.Minute)}.IsFresh(now))
}
