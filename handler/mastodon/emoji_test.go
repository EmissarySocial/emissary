package mastodon

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
)

// TestMapDocumentToEmojis_ReadsFreshAndStoredTags covers both shapes a tag arrives in: the image nested
// under "icon" as fetched from a server, and as a plain URL once stored.
func TestMapDocumentToEmojis_ReadsFreshAndStoredTags(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		"type": "Person",
		"id":   "https://example.com/users/joel",
		"tag": []any{
			map[string]any{"type": "Emoji", "name": ":casio:", "icon": map[string]any{"type": "Image", "url": "https://cdn.example.com/casio.png"}},
			map[string]any{"type": "Emoji", "name": ":stored:", "icon": "https://cdn.example.com/stored.png"},
			map[string]any{"type": "Emoji", "name": ":noimage:"},
			map[string]any{"type": "Hashtag", "name": "#go", "href": "https://example.com/tags/go"},
		},
	})

	emojis := mapDocumentToEmojis(document)

	require.Len(t, emojis, 2, "a hashtag, and an emoji with no image, are not emoji")
	require.Equal(t, "casio", emojis[0].ShortCode)
	require.Equal(t, "https://cdn.example.com/casio.png", emojis[0].URL)
	require.Equal(t, "https://cdn.example.com/casio.png", emojis[0].StaticURL)
	require.Equal(t, "stored", emojis[1].ShortCode)
	require.Equal(t, "https://cdn.example.com/stored.png", emojis[1].URL)
}

// TestMapDocumentToEmojis_NoTagsGivesAnEmptyList confirms a post with no tags yields a list, not nil.
func TestMapDocumentToEmojis_NoTagsGivesAnEmptyList(t *testing.T) {

	emojis := mapDocumentToEmojis(streams.NewDocument(map[string]any{"type": "Note", "id": "https://example.com/n/1"}))

	require.NotNil(t, emojis)
	require.Empty(t, emojis)
}

// TestMapDocumentToFields_KeepsOnlyNameValuePairs covers a profile with link fields, a file attachment, and unsafe markup.
func TestMapDocumentToFields_KeepsOnlyNameValuePairs(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		"type": "Person",
		"id":   "https://example.com/users/ben",
		"attachment": []any{
			map[string]any{"type": "PropertyValue", "name": "GitHub", "value": `<a href="https://github.com/benpate" rel="me">github.com/benpate</a>`},
			map[string]any{"type": "PropertyValue", "name": "Evil", "value": `<script>alert(1)</script>ok`},
			map[string]any{"type": "PropertyValue", "name": "", "value": "no name"},
			map[string]any{"type": "Image", "name": "banner", "url": "https://example.com/b.png"},
		},
	})

	fields := mapDocumentToFields(document)

	require.Len(t, fields, 2)
	require.Equal(t, "GitHub", fields[0].Name)
	require.Contains(t, fields[0].Value, `href="https://github.com/benpate"`)
	require.Equal(t, "Evil", fields[1].Name)
	require.NotContains(t, fields[1].Value, "<script")
}
