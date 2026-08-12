package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// hashtagStream returns a Stream carrying the provided content and one #hashtag Tag.
func hashtagStream(html string, hashtags ...string) model.Stream {

	stream := model.NewStream()
	stream.Content = model.NewContent()
	stream.Content.HTML = html

	for _, hashtag := range hashtags {
		stream.Tags = append(stream.Tags, model.NewTag(vocab.LinkTypeHashtag, hashtag))
	}

	return stream
}

// TestStream_applyHashtagLinks confirms that hashtags are linkified when the Template defines a TagURL.
func TestStream_applyHashtagLinks(t *testing.T) {
	service := &Stream{contentService: &Content{}, host: "https://example.com"}

	template := model.NewTemplate("test", nil)
	template.TagURL = "/search?q="

	stream := hashtagStream("Testing #travel here", "travel")

	service.applyHashtagLinks(&template, &stream)

	require.Equal(t, `Testing <a href="https://example.com/search?q=%23travel" target="_blank">#travel</a> here`, stream.Content.HTML, "anchors written into federated content must be absolute")
}

// TestStream_applyHashtagLinks_NoTagURL confirms that content is untouched when the Template has no TagURL.
func TestStream_applyHashtagLinks_NoTagURL(t *testing.T) {
	service := &Stream{contentService: &Content{}, host: "https://example.com"}

	template := model.NewTemplate("test", nil)
	stream := hashtagStream("Testing #travel here", "travel")

	service.applyHashtagLinks(&template, &stream)

	require.Equal(t, "Testing #travel here", stream.Content.HTML, "no TagURL means no linkification")
}

// TestStream_applyHashtagLinks_NoHashtags confirms that content is untouched when there are no hashtags.
func TestStream_applyHashtagLinks_NoHashtags(t *testing.T) {
	service := &Stream{contentService: &Content{}, host: "https://example.com"}

	template := model.NewTemplate("test", nil)
	template.TagURL = "/search?q="

	stream := hashtagStream("Testing #travel here")

	service.applyHashtagLinks(&template, &stream)

	require.Equal(t, "Testing #travel here", stream.Content.HTML, "no hashtags means nothing to link")
}

// TestStream_applyHashtagLinks_IgnoresMentions confirms that @mention Tags are not fed to the
// hashtag linkifier, which would wrap them in hashtag search links.
func TestStream_applyHashtagLinks_IgnoresMentions(t *testing.T) {
	service := &Stream{contentService: &Content{}, host: "https://example.com"}

	template := model.NewTemplate("test", nil)
	template.TagURL = "/search?q="

	stream := hashtagStream("Hey @bob@example.com about #travel", "travel")
	stream.Tags = append(stream.Tags, model.Tag{
		Type: vocab.LinkTypeMention,
		Name: "bob@example.com",
		Href: "https://example.com/@bob",
	})

	service.applyHashtagLinks(&template, &stream)

	require.Contains(t, stream.Content.HTML, `>#travel</a>`)
	require.Contains(t, stream.Content.HTML, "Hey @bob@example.com about", "mentions are not hashtag-linkified")
}
