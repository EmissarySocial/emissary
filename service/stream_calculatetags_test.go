package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// These tests exercise the hashtag pipeline at the service level -- extraction (CalculateTags) and
// linkification (applyHashtagLinks) -- with an empty SearchTag store, so NormalizeTags treats every
// tag as new and returns the parsed names unchanged. Full Stream.Save is not exercised here: it
// reaches into many concrete (non-interface) services, the same limitation noted in response_test.go.

/******************************************
 * emptyTagStore -- a data.Session whose SearchTag collection has no records
 ******************************************/

// emptyTagCollection is a data.Collection that holds no records.  Only the read methods that
// NormalizeTags reaches are implemented; the rest fail loudly if a test ever calls them.
type emptyTagCollection struct{}

func (c emptyTagCollection) Context() context.Context                              { return context.Background() }
func (c emptyTagCollection) Count(exp.Expression, ...option.Option) (int64, error) { return 0, nil }
func (c emptyTagCollection) Query(any, exp.Expression, ...option.Option) error     { return nil }
func (c emptyTagCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}
func (c emptyTagCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.NotFound("test", "unused")
}
func (c emptyTagCollection) Save(data.Object, string) error   { return derp.Internal("test", "unused") }
func (c emptyTagCollection) Delete(data.Object, string) error { return derp.Internal("test", "unused") }
func (c emptyTagCollection) HardDelete(exp.Expression) error  { return derp.Internal("test", "unused") }

// emptyTagSession is a data.Session that hands out emptyTagCollections
type emptyTagSession struct{}

func (s emptyTagSession) Collection(string) data.Collection { return emptyTagCollection{} }
func (s emptyTagSession) Context() context.Context          { return context.Background() }
func (s emptyTagSession) Close()                            {}

// newTagStreamService builds a Stream service whose one Template carries the given tag settings.
func newTagStreamService(tagPaths []string, tagURL string) (*Stream, model.Template) {

	template := model.NewTemplate("test-post", nil)
	template.TagPaths = tagPaths
	template.TagURL = tagURL

	streamService := &Stream{
		templateService:  &Template{templates: set.Map[model.Template]{"test-post": template}},
		searchTagService: &SearchTag{},
		contentService:   &Content{},
		host:             "https://example.com",
	}

	return streamService, template
}

// newTagStream returns an unsaved Stream carrying the provided HTML content
func newTagStream(html string) model.Stream {
	stream := model.NewStream()
	stream.TemplateID = "test-post"
	stream.Content = model.NewContent()
	stream.Content.HTML = html
	return stream
}

// TestStream_CalculateTags confirms that #hashtags are extracted from the schema path named in the
// Template's tagPaths. This is the direct guard against the original bug (a mis-configured path).
func TestStream_CalculateTags(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Testing hashtags #travel #Food2024 here")

	streamService.CalculateTags(emptyTagSession{}, &stream)

	// Sorted by NormalizeTags; case preserved by the parser
	require.Equal(t, []string{"Food2024", "travel"}, []string(stream.Hashtags))
}

// TestStream_CalculateTags_NoTagPaths documents that CalculateTags itself extracts nothing when the
// Template declares no tagPaths. (Stream.Save guards this at the call site so it does not clobber
// pre-existing Hashtags, but the function on its own yields an empty slice.)
func TestStream_CalculateTags_NoTagPaths(t *testing.T) {
	streamService, _ := newTagStreamService(nil, "")
	stream := newTagStream("Testing #travel here")

	streamService.CalculateTags(emptyTagSession{}, &stream)

	require.Empty(t, stream.Hashtags)
}

// TestStream_CalculateTags_DualWrites confirms that extraction fills BOTH the deprecated Hashtags
// slice and the unified Tags list. Hashtags is dual-written for one release while the external
// template packages migrate -- see projects/TAGS-UNIFICATION.md.
func TestStream_CalculateTags_DualWrites(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Testing hashtags #travel #Food2024 here")

	streamService.CalculateTags(emptyTagSession{}, &stream)

	require.Equal(t, []string{"Food2024", "travel"}, []string(stream.Hashtags))
	require.Equal(t, []string{"Food2024", "travel"}, []string(model.TagNames(stream.Tags, vocab.LinkTypeHashtag)))

	for _, tag := range stream.Tags {
		require.Equal(t, vocab.LinkTypeHashtag, tag.Type)
		require.Empty(t, tag.Href, "a hashtag's href is derived at emission, never stored")
	}
}

// TestStream_CalculateTags_PreservesMentions confirms that recalculating #hashtags leaves @mention
// entries alone. Both kinds share one field, and each is recalculated independently.
func TestStream_CalculateTags_PreservesMentions(t *testing.T) {
	streamService, _ := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Testing #travel here")
	stream.Tags = model.TagList{{
		Type: vocab.LinkTypeMention,
		Name: "sarah@mastodon.social",
		Href: "https://mastodon.social/users/sarah",
	}}

	streamService.CalculateTags(emptyTagSession{}, &stream)

	mentions := model.TagsOfType(stream.Tags, vocab.LinkTypeMention)
	require.Equal(t, 1, len(mentions), "recalculating hashtags must not disturb mentions")
	require.Equal(t, "https://mastodon.social/users/sarah", mentions[0].Href, "and must not discard their resolutions")
	require.Equal(t, []string{"travel"}, []string(model.TagNames(stream.Tags, vocab.LinkTypeHashtag)))
}

// TestStream_CalculateTags_ThenLinkify_Idempotent mirrors what Stream.Save does (extract, then
// linkify) and proves the combined pipeline is idempotent: a second pass changes nothing.
func TestStream_CalculateTags_ThenLinkify_Idempotent(t *testing.T) {
	streamService, template := newTagStreamService([]string{"content.html"}, "/search?q=")
	stream := newTagStream("Testing hashtags #travel #Food2024 here")

	// First pass -- extract + linkify
	streamService.CalculateTags(emptyTagSession{}, &stream)
	streamService.applyHashtagLinks(&template, &stream)
	first := stream.Content.HTML

	require.Equal(t, []string{"Food2024", "travel"}, []string(stream.Hashtags))
	require.Contains(t, first, `<a href="https://example.com/search?q=%23travel" target="_blank">#travel</a>`)
	require.Contains(t, first, `<a href="https://example.com/search?q=%23Food2024" target="_blank">#Food2024</a>`)

	// Second pass -- must not re-wrap or change anything
	streamService.CalculateTags(emptyTagSession{}, &stream)
	streamService.applyHashtagLinks(&template, &stream)

	require.Equal(t, []string{"Food2024", "travel"}, []string(stream.Hashtags))
	require.Equal(t, first, stream.Content.HTML, "extract+linkify must be idempotent")
}
