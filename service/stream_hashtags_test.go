package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// TestStream_HashtagAsJSONLD confirms that hashtag links use the Template's TagURL, anchored to the host.
func TestStream_HashtagAsJSONLD(t *testing.T) {
	service := &Stream{host: "https://example.com"}

	result := service.HashtagAsJSONLD("/search?q=", "travel")

	require.Equal(t, vocab.LinkTypeHashtag, result[vocab.PropertyType])
	require.Equal(t, "#travel", result[vocab.PropertyName], "Mastodon convention includes the # prefix")
	require.Equal(t, "https://example.com/search?q=%23travel", result[vocab.PropertyHref], "the path must come from the Template")
}

// TestStream_HashtagAsJSONLD_OtherTagURL confirms that a Template can point its hashtags somewhere else.
func TestStream_HashtagAsJSONLD_OtherTagURL(t *testing.T) {
	service := &Stream{host: "https://example.com"}

	result := service.HashtagAsJSONLD("/home?q=", "travel")

	require.Equal(t, "https://example.com/home?q=%23travel", result[vocab.PropertyHref])
}

// TestStream_HashtagAsJSONLD_NoTagURL confirms that hashtags are published without a link when the Template has no TagURL.
func TestStream_HashtagAsJSONLD_NoTagURL(t *testing.T) {
	service := &Stream{host: "https://example.com"}

	result := service.HashtagAsJSONLD("", "travel")

	require.Equal(t, vocab.LinkTypeHashtag, result[vocab.PropertyType])
	require.Equal(t, "#travel", result[vocab.PropertyName])
	require.NotContains(t, result, vocab.PropertyHref, "no TagURL means no link")
}

// TestStream_applyHashtagLinks confirms that hashtags are linkified when the Template defines a TagURL.
func TestStream_applyHashtagLinks(t *testing.T) {
	service := &Stream{contentService: &Content{}, host: "https://example.com"}

	template := model.NewTemplate("test", nil)
	template.TagURL = "/search?q="

	stream := model.NewStream()
	stream.Content = model.NewContent()
	stream.Content.HTML = "Testing #travel here"
	stream.Hashtags = []string{"travel"}

	service.applyHashtagLinks(&template, &stream)

	require.Equal(t, `Testing <a href="https://example.com/search?q=%23travel" target="_blank">#travel</a> here`, stream.Content.HTML, "anchors written into federated content must be absolute")
}

// TestStream_applyHashtagLinks_NoTagURL confirms that content is untouched when the Template has no TagURL.
func TestStream_applyHashtagLinks_NoTagURL(t *testing.T) {
	service := &Stream{contentService: &Content{}, host: "https://example.com"}

	template := model.NewTemplate("test", nil)

	stream := model.NewStream()
	stream.Content = model.NewContent()
	stream.Content.HTML = "Testing #travel here"
	stream.Hashtags = []string{"travel"}

	service.applyHashtagLinks(&template, &stream)

	require.Equal(t, "Testing #travel here", stream.Content.HTML, "no TagURL means no linkification")
}

// TestStream_applyHashtagLinks_NoHashtags confirms that content is untouched when there are no hashtags.
func TestStream_applyHashtagLinks_NoHashtags(t *testing.T) {
	service := &Stream{contentService: &Content{}, host: "https://example.com"}

	template := model.NewTemplate("test", nil)
	template.TagURL = "/search?q="

	stream := model.NewStream()
	stream.Content = model.NewContent()
	stream.Content.HTML = "Testing #travel here"

	service.applyHashtagLinks(&template, &stream)

	require.Equal(t, "Testing #travel here", stream.Content.HTML, "no hashtags means nothing to link")
}
