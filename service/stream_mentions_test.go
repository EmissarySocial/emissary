package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// These tests cover the @mention half of the tag pipeline (BUG-09): extraction into Stream.Tags,
// and emission of `Mention` tags plus `cc` addressing. They reuse the harness in
// stream_calculatetags_test.go.
//
// Resolution itself is NOT exercised: it calls out to WebFinger through the ActivityStream client
// stack, which the service-level harness cannot stand up (the same limitation noted in
// response_test.go). These tests set Hrefs directly to stand in for a completed lookup, and the
// service under test has a nil activityService -- so every case here must extract handles that are
// either already resolved or, where a lookup would fire, is asserted to be absent.

// mentionsOf returns the Mention-typed Tags on a Stream.
func mentionsOf(stream model.Stream) model.TagList {
	return model.TagsOfType(stream.Tags, vocab.LinkTypeMention)
}

/******************************************
 * Extraction
 ******************************************/

// TestStream_CalculateMentions_PreservesResolved is the load-bearing case: extraction re-runs on
// EVERY save, so a handle that has already been resolved must keep its Href.
func TestStream_CalculateMentions_PreservesResolved(t *testing.T) {

	// Without this, editing a post would discard its resolutions and spend a fresh WebFinger
	// lookup per handle on the next save. It is also what keeps this test off the network.
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Hey @sarah@mastodon.social")
	stream.Tags = model.TagList{{
		Type: vocab.LinkTypeMention,
		Name: "sarah@mastodon.social",
		Href: "https://mastodon.social/users/sarah",
	}}

	streamService.CalculateMentions(&stream)

	mentions := mentionsOf(stream)
	require.Equal(t, 1, len(mentions))
	require.Equal(t, "sarah@mastodon.social", mentions[0].Name)
	require.Equal(t, "https://mastodon.social/users/sarah", mentions[0].Href, "existing resolution must survive a re-scan")
}

// TestStream_CalculateMentions_PreservesUnresolvableSentinel confirms the negative cache: a handle
// already marked unresolvable is NOT looked up again. Without this, a dead handle would hit the
// network on every single save, forever -- and dead handles are the slowest lookups there are.
func TestStream_CalculateMentions_PreservesUnresolvableSentinel(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Hey @nobody@nowhere.invalid")
	stream.Tags = model.TagList{{
		Type: vocab.LinkTypeMention,
		Name: "nobody@nowhere.invalid",
		Href: model.TagHrefUnresolvable,
	}}

	streamService.CalculateMentions(&stream)

	mentions := mentionsOf(stream)
	require.Equal(t, 1, len(mentions))
	require.Equal(t, model.TagHrefUnresolvable, mentions[0].Href, "a failed lookup must not be retried")
	require.False(t, mentions[0].IsResolved(), "and must never be federated")
}

// TestStream_CalculateMentions_DropsRemovedHandle confirms that deleting a mention from the content
// removes it from the Stream, resolved or not.
func TestStream_CalculateMentions_DropsRemovedHandle(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Never mind")
	stream.Tags = model.TagList{{
		Type: vocab.LinkTypeMention,
		Name: "sarah@mastodon.social",
		Href: "https://mastodon.social/users/sarah",
	}}

	streamService.CalculateMentions(&stream)

	require.Empty(t, mentionsOf(stream))
}

// TestStream_CalculateMentions_PreservesHashtags confirms that recalculating mentions leaves
// #hashtag entries untouched -- the two kinds share one field and are calculated separately.
func TestStream_CalculateMentions_PreservesHashtags(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Nothing to mention here, just #travel")
	stream.Tags = model.TagList{model.NewTag(vocab.LinkTypeHashtag, "travel")}

	streamService.CalculateMentions(&stream)

	require.Equal(t, []string{"travel"}, []string(model.TagNames(stream.Tags, vocab.LinkTypeHashtag)))
}

// TestStream_CalculateMentions_NoTagPaths documents that nothing is extracted when the Template
// declares no tagPaths. Stream.Save guards this at the call site.
func TestStream_CalculateMentions_NoTagPaths(t *testing.T) {
	streamService, _ := newTagStreamService(nil, "")
	stream := newTagStream("Hey @sarah@mastodon.social")

	streamService.CalculateMentions(&stream)

	require.Empty(t, mentionsOf(stream))
}

// TestStream_CalculateMentions_SkipsEmptyTokens guards the parser quirk that a bare "@" followed by
// whitespace yields an empty token. Without the filter, resolution would spend a WebFinger lookup
// on the empty string.
func TestStream_CalculateMentions_SkipsEmptyTokens(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Doors @ 8pm, meet @ the bar")

	streamService.CalculateMentions(&stream)

	require.Empty(t, mentionsOf(stream))
}

// TestStream_CalculateMentions_IgnoresEmailAddresses confirms that an email address in the content
// is not mistaken for a mention. The parser only opens a token at a prefix that follows whitespace,
// so the "@" inside "ben@pate.org" never starts one.
func TestStream_CalculateMentions_IgnoresEmailAddresses(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Write to ben@pate.org for details")

	streamService.CalculateMentions(&stream)

	require.Empty(t, mentionsOf(stream))
}

// TestStream_CalculateMentions_BareHandleUsesLocalDomain confirms that a handle with no hostname is
// anchored to this server, and that the bare and qualified forms collapse into one Tag. Both are
// pre-resolved here so no lookup fires.
func TestStream_CalculateMentions_BareHandleUsesLocalDomain(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Hey @bob -- and again, @bob@example.com")
	stream.Tags = model.TagList{{
		Type: vocab.LinkTypeMention,
		Name: "bob@example.com",
		Href: "https://example.com/@bob",
	}}

	streamService.CalculateMentions(&stream)

	mentions := mentionsOf(stream)
	require.Equal(t, 1, len(mentions), "the bare and qualified forms are the same person")
	require.Equal(t, "bob@example.com", mentions[0].Name)
	require.Equal(t, "@bob@example.com", mentions[0].DisplayName(), "the published name must be unambiguous to remote readers")
}

// TestStream_CalculateMentions_BareHandleNoHost confirms that a bare handle is DROPPED when the
// service has no hostname to anchor it to, rather than stored in a form that can never resolve.
func TestStream_CalculateMentions_BareHandleNoHost(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "")
	streamService.host = ""
	stream := newTagStream("Hey @bob")

	streamService.CalculateMentions(&stream)

	require.Empty(t, mentionsOf(stream))
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
	stream.Tags = model.TagList{{
		Type: vocab.LinkTypeMention,
		Name: "sarah@mastodon.social",
		Href: "https://mastodon.social/users/sarah",
	}}

	document := streamService.JSONLD(emptyTagSession{}, &stream)

	require.Equal(t, []mapof.String{{
		vocab.PropertyType: vocab.LinkTypeMention,
		vocab.PropertyName: "@sarah@mastodon.social",
		vocab.PropertyHref: "https://mastodon.social/users/sarah",
	}}, tagsOf(t, document))

	require.Equal(t, []string{"https://mastodon.social/users/sarah"}, document[vocab.PropertyCC])
}

// TestStream_JSONLD_HashtagsAndMentions confirms the two tag kinds coexist. The `tag` property is
// assigned exactly once, so this is the guard against one kind clobbering the other. It also pins
// the href rule: the hashtag's link is DERIVED from the Template, the mention's is STORED.
func TestStream_JSONLD_HashtagsAndMentions(t *testing.T) {
	streamService := newMentionStreamService()
	stream := newTagStream("Hey @sarah@mastodon.social about #travel")
	stream.Tags = model.TagList{
		model.NewTag(vocab.LinkTypeHashtag, "travel"),
		{Type: vocab.LinkTypeMention, Name: "sarah@mastodon.social", Href: "https://mastodon.social/users/sarah"},
	}

	tags := tagsOf(t, streamService.JSONLD(emptyTagSession{}, &stream))

	require.Equal(t, 2, len(tags), "hashtags and mentions must both survive")

	require.Equal(t, vocab.LinkTypeHashtag, tags[0][vocab.PropertyType])
	require.Equal(t, "#travel", tags[0][vocab.PropertyName])
	require.Equal(t, "https://example.com/search?q=%23travel", tags[0][vocab.PropertyHref], "derived from the Template's tagUrl")

	require.Equal(t, vocab.LinkTypeMention, tags[1][vocab.PropertyType])
	require.Equal(t, "@sarah@mastodon.social", tags[1][vocab.PropertyName])
	require.Equal(t, "https://mastodon.social/users/sarah", tags[1][vocab.PropertyHref], "stored, because it cannot be derived")
}

// TestStream_JSONLD_HashtagHrefFollowsTemplate is the reason hashtag hrefs are NOT stored: changing
// a Template's tagUrl must take effect on every existing document immediately, not only on the ones
// that happen to be re-saved afterwards.
func TestStream_JSONLD_HashtagHrefFollowsTemplate(t *testing.T) {
	streamService, template := newTagStreamService([]string{"content.html"}, "/search?q=")
	streamService.attachmentService = &Attachment{}

	stream := newTagStream("About #travel")
	stream.Tags = model.TagList{model.NewTag(vocab.LinkTypeHashtag, "travel")}

	before := tagsOf(t, streamService.JSONLD(emptyTagSession{}, &stream))
	require.Equal(t, "https://example.com/search?q=%23travel", before[0][vocab.PropertyHref])

	// The operator repoints the Template. The Stream is NOT re-saved.
	template.TagURL = "/albums?q="
	streamService.templateService.templates["test-post"] = template

	after := tagsOf(t, streamService.JSONLD(emptyTagSession{}, &stream))
	require.Equal(t, "https://example.com/albums?q=%23travel", after[0][vocab.PropertyHref], "a stored href would still point at the old URL")
}

// TestStream_JSONLD_UnresolvedMentionNotPublished confirms that a handle which never resolved --
// and one that was marked unresolvable -- are neither tagged nor addressed.
func TestStream_JSONLD_UnresolvedMentionNotPublished(t *testing.T) {
	streamService := newMentionStreamService()
	stream := newTagStream("Hey @nobody@nowhere.invalid @pending@example.org and @sarah@mastodon.social")
	stream.Tags = model.TagList{
		{Type: vocab.LinkTypeMention, Name: "nobody@nowhere.invalid", Href: model.TagHrefUnresolvable},
		{Type: vocab.LinkTypeMention, Name: "pending@example.org"},
		{Type: vocab.LinkTypeMention, Name: "sarah@mastodon.social", Href: "https://mastodon.social/users/sarah"},
	}

	document := streamService.JSONLD(emptyTagSession{}, &stream)
	tags := tagsOf(t, document)

	require.Equal(t, 1, len(tags))
	require.Equal(t, "@sarah@mastodon.social", tags[0][vocab.PropertyName])
	require.Equal(t, []string{"https://mastodon.social/users/sarah"}, document[vocab.PropertyCC])
}

// TestStream_JSONLD_NoTags confirms that a document with no tags omits `tag` and `cc` entirely,
// rather than publishing empty arrays.
func TestStream_JSONLD_NoTags(t *testing.T) {
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
	stream.Tags = model.TagList{{
		Type: vocab.LinkTypeMention,
		Name: "sarah@mastodon.social",
		Href: "https://mastodon.social/users/sarah",
	}}

	document := streamService.JSONLD(emptyTagSession{}, &stream)

	cc, ok := document[vocab.PropertyCC].([]string)
	require.True(t, ok, "cc must be a plain []string")

	// Reproduce exactly what publish_outbox does with it
	cc = append(cc, "https://example.com/@author")
	require.Equal(t, []string{"https://mastodon.social/users/sarah", "https://example.com/@author"}, cc)
}
