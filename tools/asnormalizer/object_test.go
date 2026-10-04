package asnormalizer

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
)

func hashtagNote() map[string]any {
	return map[string]any{
		"type":    "Note",
		"id":      "https://example.com/notes/1",
		"content": "hello #cats",
		"tag": []any{
			map[string]any{"type": "Hashtag", "href": "https://example.com/tags/cats", "name": "#cats"},
		},
	}
}

// A post's tags belong to the post itself, so they must survive when the post
// arrives wrapped in a Create.
func TestObject_KeepsTagsOfWrappedPost(t *testing.T) {

	wrapper := streams.NewDocument(map[string]any{
		"type":   "Create",
		"actor":  "https://example.com/users/alice",
		"object": hashtagNote(),
	})

	tags, ok := Object(nil, wrapper)["tag"].([]map[string]any)

	require.True(t, ok)
	require.Len(t, tags, 1)
	require.Equal(t, "Hashtag", tags[0]["type"])
	require.Equal(t, "#cats", tags[0]["name"])
	require.Equal(t, "https://example.com/tags/cats", tags[0]["href"])
}

func TestObject_KeepsTagsOfUnwrappedPost(t *testing.T) {

	tags, ok := Object(nil, streams.NewDocument(hashtagNote()))["tag"].([]map[string]any)

	require.True(t, ok)
	require.Len(t, tags, 1)
}

// emojiNote is a post whose tags include a custom emoji, a javascript: emoji, and a hashtag that carries an icon.
func emojiNote() map[string]any {
	return map[string]any{
		"type":    "Note",
		"id":      "https://example.com/notes/2",
		"content": "hello :blobcat:",
		"tag": []any{
			map[string]any{"type": "Emoji", "id": "https://example.com/emojis/1", "name": ":blobcat:",
				"icon": map[string]any{"type": "Image", "url": "https://cdn.example.com/blobcat.png"}},
			map[string]any{"type": "Emoji", "id": "https://example.com/emojis/2", "name": ":bad:",
				"icon": map[string]any{"type": "Image", "url": "javascript:alert(1)"}},
			map[string]any{"type": "Hashtag", "href": "https://example.com/tags/cats", "name": "#cats",
				"icon": map[string]any{"type": "Image", "url": "https://cdn.example.com/ignored.png"}},
		},
	}
}

// TestObject_KeepsTheImageOfACustomEmoji confirms an emoji tag keeps its image URL, and that nothing else does.
func TestObject_KeepsTheImageOfACustomEmoji(t *testing.T) {

	tags, ok := Object(nil, streams.NewDocument(emojiNote()))["tag"].([]map[string]any)

	require.True(t, ok)
	require.Len(t, tags, 3)

	require.Equal(t, "https://cdn.example.com/blobcat.png", tags[0]["icon"])
	require.Equal(t, "https://example.com/emojis/1", tags[0]["href"])
	require.NotContains(t, tags[1], "icon", "an emoji image that is not a web address is dropped")
	require.NotContains(t, tags[2], "icon", "only emoji keep an image")
}

// TestObject_KeepsEditTimeTotalsAndAttachmentNames confirms the properties the Mastodon API reads survive ingest.
func TestObject_KeepsEditTimeTotalsAndAttachmentNames(t *testing.T) {

	note := streams.NewDocument(map[string]any{
		"type":       "Note",
		"id":         "https://example.com/notes/3",
		"content":    "hello",
		"updated":    "2026-10-01T12:30:00Z",
		"likes":      map[string]any{"type": "Collection", "totalItems": 7},
		"shares":     map[string]any{"type": "Collection", "totalItems": 3},
		"attachment": []any{map[string]any{"type": "Image", "url": "https://cdn.example.com/a.png", "name": "A cat on a sofa"}},
	})

	result := Object(nil, note)

	require.NotNil(t, result["updated"])
	require.Equal(t, map[string]any{"totalItems": int64(7)}, result["likes"])
	require.Equal(t, map[string]any{"totalItems": int64(3)}, result["shares"])

	attachments, ok := result["attachment"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, "A cat on a sofa", attachments[0]["name"])

	// A post that reports nothing stores nothing
	plain := Object(nil, streams.NewDocument(map[string]any{"type": "Note", "id": "https://example.com/notes/4", "content": "hi"}))
	require.NotContains(t, plain, "likes")
	require.NotContains(t, plain, "shares")
	require.NotContains(t, plain, "updated")
}
