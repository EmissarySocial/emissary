package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// These tests cover the @mention half of the tag pipeline (BUG-09): extraction (CalculateMentions),
// and emission of `Mention` tags plus `cc` addressing (JSONLD). They reuse the harness in
// stream_calculatetags_test.go. Resolution itself (resolveMentions) is not exercised here -- it
// makes live WebFinger calls through the ActivityStream client stack -- so these tests set Hrefs
// directly to stand in for a completed resolution.

/******************************************
 * Extraction
 ******************************************/

// TestStream_CalculateMentions confirms that @handles are extracted from the schema path named in
// the Template's tagPaths, and that they arrive UNRESOLVED (no Href) -- resolution is a separate,
// publish-time step.
func TestStream_CalculateMentions(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Hey @sarah@mastodon.social and @bob@example.org, look at this")

	streamService.CalculateMentions(&stream)

	require.Equal(t, 2, len(stream.Mentions))
	require.Equal(t, "sarah@mastodon.social", stream.Mentions[0].Handle)
	require.Equal(t, "bob@example.org", stream.Mentions[1].Handle)

	for _, mention := range stream.Mentions {
		require.True(t, mention.NotResolved(), "extraction must not resolve")
		require.Equal(t, "", mention.Href)
	}
}

// TestStream_CalculateMentions_NoTagPaths documents that CalculateMentions extracts nothing when
// the Template declares no tagPaths. Stream.Save guards this at the call site.
func TestStream_CalculateMentions_NoTagPaths(t *testing.T) {
	streamService, _ := newTagStreamService(nil, "")
	stream := newTagStream("Hey @sarah@mastodon.social")

	streamService.CalculateMentions(&stream)

	require.Empty(t, stream.Mentions)
}

// TestStream_CalculateMentions_PreservesResolved is the load-bearing case: extraction re-runs
// on EVERY save, so a handle that has already been resolved must keep its Href.
func TestStream_CalculateMentions_PreservesResolved(t *testing.T) {

	// Without this, editing a post would discard its resolutions and spend a fresh WebFinger
	// lookup per handle on the next publish.

	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Hey @sarah@mastodon.social")

	// First pass -- extract, then stand in for a completed resolution
	streamService.CalculateMentions(&stream)
	require.Equal(t, 1, len(stream.Mentions))
	stream.Mentions[0].Href = "https://mastodon.social/users/sarah"

	// The author edits the post, adding a second mention. Saving re-scans.
	stream.Content.HTML = "Hey @sarah@mastodon.social and @bob@example.org"
	streamService.CalculateMentions(&stream)

	require.Equal(t, 2, len(stream.Mentions))

	require.Equal(t, "sarah@mastodon.social", stream.Mentions[0].Handle)
	require.Equal(t, "https://mastodon.social/users/sarah", stream.Mentions[0].Href, "existing resolution must survive a re-scan")

	require.Equal(t, "bob@example.org", stream.Mentions[1].Handle)
	require.True(t, stream.Mentions[1].NotResolved(), "the newly-added handle is not resolved yet")
}

// TestStream_CalculateMentions_DropsRemovedHandle confirms that deleting a mention from the content
// removes it from the Stream, resolved or not.
func TestStream_CalculateMentions_DropsRemovedHandle(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Hey @sarah@mastodon.social")

	streamService.CalculateMentions(&stream)
	stream.Mentions[0].Href = "https://mastodon.social/users/sarah"

	stream.Content.HTML = "Never mind"
	streamService.CalculateMentions(&stream)

	require.Empty(t, stream.Mentions)
}

// TestStream_CalculateMentions_SkipsEmptyTokens guards the parser quirk that a bare "@" followed by
// whitespace yields an empty token. Without the filter, resolveMentions would spend a WebFinger
// lookup on the empty string.
func TestStream_CalculateMentions_SkipsEmptyTokens(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Doors @ 8pm, meet @ the bar")

	streamService.CalculateMentions(&stream)

	require.Empty(t, stream.Mentions)
}

// TestStream_CalculateMentions_BareHandleUsesLocalDomain confirms that a handle with no hostname
// is anchored to this server -- on example.com, "@bob" means "@bob@example.com".
func TestStream_CalculateMentions_BareHandleUsesLocalDomain(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Hey @bob, take a look")

	streamService.CalculateMentions(&stream)

	require.Equal(t, 1, len(stream.Mentions))
	require.Equal(t, "bob@example.com", stream.Mentions[0].Handle)
	require.Equal(t, "@bob@example.com", stream.Mentions[0].Name(), "the published name must be unambiguous to remote readers")
}

// TestStream_CalculateMentions_BareAndQualifiedDedupe confirms that the bare and fully-qualified
// forms of one local handle collapse into a single Mention.
func TestStream_CalculateMentions_BareAndQualifiedDedupe(t *testing.T) {

	// This is why qualification happens at extraction rather than at resolution -- deduping
	// after the fact would require re-deriving the local form of every handle.
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Hey @bob -- and again, @bob@example.com")

	streamService.CalculateMentions(&stream)

	require.Equal(t, 1, len(stream.Mentions))
	require.Equal(t, "bob@example.com", stream.Mentions[0].Handle)
}

// TestStream_CalculateMentions_BareHandleNoHost confirms that a bare handle is DROPPED when the
// service has no hostname to anchor it to, rather than stored in a form that can never resolve.
func TestStream_CalculateMentions_BareHandleNoHost(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "")
	streamService.host = ""
	stream := newTagStream("Hey @bob and @sarah@mastodon.social")

	streamService.CalculateMentions(&stream)

	require.Equal(t, 1, len(stream.Mentions))
	require.Equal(t, "sarah@mastodon.social", stream.Mentions[0].Handle)
}

// TestStream_CalculateMentions_BareTokenFalsePositive documents a known cost of anchoring bare
// handles: any "@word" is treated as a local handle, and only fails later, at resolution.
func TestStream_CalculateMentions_BareTokenFalsePositive(t *testing.T) {

	// "@8pm" is not a person, but nothing at extraction time can tell. It becomes
	// "8pm@example.com", fails WebFinger at publish, and is reported and skipped there.
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Doors @8pm sharp")

	streamService.CalculateMentions(&stream)

	require.Equal(t, 1, len(stream.Mentions))
	require.Equal(t, "8pm@example.com", stream.Mentions[0].Handle)
	require.True(t, stream.Mentions[0].NotResolved())
}

// TestStream_CalculateMentions_Deduplicates confirms that mentioning one person several times
// produces a single Mention (and therefore a single lookup, tag, and cc entry).
func TestStream_CalculateMentions_Deduplicates(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("@sarah@mastodon.social hi. @sarah@mastodon.social again. @sarah@mastodon.social")

	streamService.CalculateMentions(&stream)

	require.Equal(t, 1, len(stream.Mentions))
	require.Equal(t, "sarah@mastodon.social", stream.Mentions[0].Handle)
}

// TestStream_CalculateMentions_IgnoresEmailAddresses confirms that an email address in the content
// is not mistaken for a mention. The parser only opens a token at a prefix that follows whitespace,
// so the "@" inside "ben@pate.org" never starts one.
func TestStream_CalculateMentions_IgnoresEmailAddresses(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Write to ben@pate.org for details")

	streamService.CalculateMentions(&stream)

	require.Empty(t, stream.Mentions)
}

// TestStream_CalculateMentions_MultiplePaths confirms that every configured tagPath is scanned.
func TestStream_CalculateMentions_MultiplePaths(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"label", "content.html"}, "/search?q=")
	stream := newTagStream("and @bob@example.org")
	stream.Label = "A note for @sarah@mastodon.social"

	streamService.CalculateMentions(&stream)

	require.Equal(t, 2, len(stream.Mentions))
	require.Equal(t, "sarah@mastodon.social", stream.Mentions[0].Handle)
	require.Equal(t, "bob@example.org", stream.Mentions[1].Handle)
}

/******************************************
 * Emission
 ******************************************/

// newMentionStreamService extends the tag harness with the services JSONLD needs.
func newMentionStreamService() *Stream {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	streamService.attachmentService = &Attachment{}
	return streamService
}

// tagsOf returns the `tag` array from a JSON-LD document, or nil when it has none.
func tagsOf(t *testing.T, document mapof.Any) []mapof.String {
	t.Helper()

	value, exists := document[vocab.PropertyTag]

	if !exists {
		return nil
	}

	tags, ok := value.([]mapof.String)
	require.True(t, ok, "tag array must be []mapof.String")
	return tags
}

// TestStream_JSONLD_Mentions is the direct guard against BUG-09: a published document must carry a
// `Mention` tag for each resolved handle, AND address that actor in `cc` -- without the addressing,
// a mentioned account that does not follow the author never receives the post at all.
func TestStream_JSONLD_Mentions(t *testing.T) {
	streamService := newMentionStreamService()
	stream := newTagStream("Hey @sarah@mastodon.social")
	stream.Mentions = model.NewMentions()
	stream.Mentions = append(stream.Mentions, model.Mention{
		Handle: "sarah@mastodon.social",
		Href:   "https://mastodon.social/users/sarah",
	})

	document := streamService.JSONLD(emptyTagSession{}, &stream)

	require.Equal(t, []mapof.String{{
		vocab.PropertyType: vocab.LinkTypeMention,
		vocab.PropertyName: "@sarah@mastodon.social",
		vocab.PropertyHref: "https://mastodon.social/users/sarah",
	}}, tagsOf(t, document))

	require.Equal(t, []string{"https://mastodon.social/users/sarah"}, document[vocab.PropertyCC])
}

// TestStream_JSONLD_HashtagsAndMentions confirms the two tag kinds coexist. The `tag` property is
// assigned exactly once, so this is the guard against one kind clobbering the other.
func TestStream_JSONLD_HashtagsAndMentions(t *testing.T) {
	streamService := newMentionStreamService()
	stream := newTagStream("Hey @sarah@mastodon.social about #travel")
	stream.Hashtags = []string{"travel"}
	stream.Mentions = append(model.NewMentions(), model.Mention{
		Handle: "sarah@mastodon.social",
		Href:   "https://mastodon.social/users/sarah",
	})

	tags := tagsOf(t, streamService.JSONLD(emptyTagSession{}, &stream))

	require.Equal(t, 2, len(tags), "hashtags and mentions must both survive")
	require.Equal(t, vocab.LinkTypeHashtag, tags[0][vocab.PropertyType])
	require.Equal(t, "#travel", tags[0][vocab.PropertyName])
	require.Equal(t, vocab.LinkTypeMention, tags[1][vocab.PropertyType])
	require.Equal(t, "@sarah@mastodon.social", tags[1][vocab.PropertyName])
}

// TestStream_JSONLD_UnresolvedMentionNotPublished confirms that a handle which never resolved is
// neither tagged nor addressed. A `Mention` with no `href` gives a receiving server nothing to
// route to, and publishing must not fail just because a handle was mistyped.
func TestStream_JSONLD_UnresolvedMentionNotPublished(t *testing.T) {
	streamService := newMentionStreamService()
	stream := newTagStream("Hey @nobody@nowhere.invalid and @sarah@mastodon.social")
	stream.Mentions = append(model.NewMentions(),
		model.Mention{Handle: "nobody@nowhere.invalid"}, // never resolved
		model.Mention{Handle: "sarah@mastodon.social", Href: "https://mastodon.social/users/sarah"},
	)

	document := streamService.JSONLD(emptyTagSession{}, &stream)
	tags := tagsOf(t, document)

	require.Equal(t, 1, len(tags))
	require.Equal(t, "@sarah@mastodon.social", tags[0][vocab.PropertyName])
	require.Equal(t, []string{"https://mastodon.social/users/sarah"}, document[vocab.PropertyCC])
}

// TestStream_JSONLD_NoMentions confirms that a document with no tags of either kind omits `tag` and
// `cc` entirely, rather than publishing empty arrays.
func TestStream_JSONLD_NoMentions(t *testing.T) {
	streamService := newMentionStreamService()
	stream := newTagStream("Nothing to see here")

	document := streamService.JSONLD(emptyTagSession{}, &stream)

	require.NotContains(t, document, vocab.PropertyTag)
	require.NotContains(t, document, vocab.PropertyCC)
}

// TestStream_JSONLD_CCIsPlainStringSlice pins the concrete type of `cc`. service.Stream.publish_outbox
// type-asserts it as []string when appending the in-reply-to author; a named slice type would fail
// that assertion silently and drop the reply author from delivery.
func TestStream_JSONLD_CCIsPlainStringSlice(t *testing.T) {
	streamService := newMentionStreamService()
	stream := newTagStream("Hey @sarah@mastodon.social")
	stream.Mentions = append(model.NewMentions(), model.Mention{
		Handle: "sarah@mastodon.social",
		Href:   "https://mastodon.social/users/sarah",
	})

	document := streamService.JSONLD(emptyTagSession{}, &stream)

	cc, ok := document[vocab.PropertyCC].([]string)
	require.True(t, ok, "cc must be a plain []string")

	// Reproduce exactly what publish_outbox does with it
	cc = append(cc, "https://example.com/@author")
	require.Equal(t, []string{"https://mastodon.social/users/sarah", "https://example.com/@author"}, cc)
}
