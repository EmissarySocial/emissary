package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// TestTag_DisplayName confirms that each tag type carries its own microsyntax prefix into AS2.
func TestTag_DisplayName(t *testing.T) {
	require.Equal(t, "#Food2024", NewTag(vocab.LinkTypeHashtag, "Food2024").DisplayName())
	require.Equal(t, "@bob@server.social", NewTag(vocab.LinkTypeMention, "bob@server.social").DisplayName())
	require.Equal(t, "unknown", Tag{Type: "SomethingElse", Name: "unknown"}.DisplayName())
}

// TestTag_JSONLD confirms the AS2 shape, and that a tag with no link target is published WITHOUT
// an href rather than with an empty one -- a Template that defines no tagUrl means
// "extract, but do not linkify".
func TestTag_JSONLD(t *testing.T) {

	tag := NewTag(vocab.LinkTypeHashtag, "travel")

	withHref := tag.JSONLD("https://example.com", "/search?q=")
	require.Equal(t, vocab.LinkTypeHashtag, withHref[vocab.PropertyType])
	require.Equal(t, "#travel", withHref[vocab.PropertyName])
	require.Equal(t, "https://example.com/search?q=%23travel", withHref[vocab.PropertyHref])

	// An empty tagUrl means "extract, but do not linkify"
	withoutHref := tag.JSONLD("https://example.com", "")
	require.Equal(t, "#travel", withoutHref[vocab.PropertyName])
	require.NotContains(t, withoutHref, vocab.PropertyHref, "no link target means no href")
}

// TestTag_Resolution covers the three states a Mention passes through: never looked up, looked up
// and found, looked up and unresolvable.
func TestTag_Resolution(t *testing.T) {

	pending := NewTag(vocab.LinkTypeMention, "bob@server.social")
	require.True(t, pending.NeedsResolution())
	require.False(t, pending.IsResolved())

	found := Tag{Type: vocab.LinkTypeMention, Name: "bob@server.social", Href: "https://server.social/@bob"}
	require.False(t, found.NeedsResolution())
	require.True(t, found.IsResolved())

	// The sentinel stops the lookup from being retried on every save, while keeping the tag
	// out of the federated document.
	dead := Tag{Type: vocab.LinkTypeMention, Name: "nobody@nowhere.invalid", Href: TagHrefUnresolvable}
	require.False(t, dead.NeedsResolution(), "a failed lookup must not be retried")
	require.False(t, dead.IsResolved(), "a failed lookup must never be federated")
}

// TestTag_HashtagNeverNeedsResolution confirms that hashtags are exempt: their href is derived at
// emission, so they must never enter the resolution queue.
func TestTag_HashtagNeverNeedsResolution(t *testing.T) {
	require.False(t, NewTag(vocab.LinkTypeHashtag, "travel").NeedsResolution())
}

// TestTag_Link pins the derived-vs-stored rule that the whole design turns on: a #hashtag's target
// is rebuilt from the host and Template on every call, while an @mention's is read from storage.
func TestTag_Link(t *testing.T) {

	hashtag := NewTag(vocab.LinkTypeHashtag, "travel")
	require.Equal(t, "https://example.com/search?q=%23travel", hashtag.Link("https://example.com", "/search?q="))
	require.Equal(t, "https://other.social/search?q=%23travel", hashtag.Link("https://other.social", "/search?q="), "derived from the host it is asked about")
	require.Equal(t, "https://example.com/albums?q=%23travel", hashtag.Link("https://example.com", "/albums?q="), "and from the Template it is asked about")
	require.Equal(t, "", hashtag.Link("https://example.com", ""), "no tagUrl means no link")

	// A mention ignores the hashtag arguments entirely -- its target cannot be derived from them.
	resolved := Tag{Type: vocab.LinkTypeMention, Name: "bob@server.social", Href: "https://server.social/@bob"}
	require.Equal(t, "https://server.social/@bob", resolved.Link("https://example.com", "/search?q="))

	require.Equal(t, "", NewTag(vocab.LinkTypeMention, "bob@server.social").Link("https://example.com", "/search?q="), "unresolved has no link")

	dead := Tag{Type: vocab.LinkTypeMention, Name: "nobody@nowhere.invalid", Href: TagHrefUnresolvable}
	require.Equal(t, "", dead.Link("https://example.com", "/search?q="), "the sentinel is not a link")
}

// TestTag_GetPointer covers every branch of the schema accessor, including the unknown-name miss --
// the path schema traversal takes when a caller asks for a property that does not exist.
func TestTag_GetPointer(t *testing.T) {

	tag := Tag{Type: vocab.LinkTypeMention, Name: "bob@server.social", Href: "https://server.social/@bob"}

	pointer, ok := tag.GetPointer("type")
	require.True(t, ok)
	require.Equal(t, &tag.Type, pointer)

	pointer, ok = tag.GetPointer("name")
	require.True(t, ok)
	require.Equal(t, &tag.Name, pointer)

	pointer, ok = tag.GetPointer("href")
	require.True(t, ok)
	require.Equal(t, &tag.Href, pointer)

	pointer, ok = tag.GetPointer("nope")
	require.False(t, ok, "an unknown property must not resolve")
	require.Nil(t, pointer)
}

// TestTag_GetPointer_Writable confirms the pointers actually mutate the Tag -- the property that
// makes schema.Set work at all.
func TestTag_GetPointer_Writable(t *testing.T) {

	tag := NewTag(vocab.LinkTypeHashtag, "travel")

	pointer, ok := tag.GetPointer("name")
	require.True(t, ok)

	*(pointer.(*string)) = "coffee"
	require.Equal(t, "coffee", tag.Name)
}
