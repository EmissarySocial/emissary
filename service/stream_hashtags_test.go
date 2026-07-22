package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestStream_applyHashtagLinks confirms that hashtags are linkified when the Template defines a TagURL.
func TestStream_applyHashtagLinks(t *testing.T) {
	service := &Stream{contentService: &Content{}}

	template := model.NewTemplate("test", nil)
	template.TagURL = "/search?q="

	stream := model.NewStream()
	stream.Content = model.NewContent()
	stream.Content.HTML = "Testing #travel here"
	stream.Hashtags = []string{"travel"}

	service.applyHashtagLinks(&template, &stream)

	require.Equal(t, `Testing <a href="/search?q=%23travel" target="_blank">#travel</a> here`, stream.Content.HTML)
}

// TestStream_applyHashtagLinks_NoTagURL confirms that content is untouched when the Template has no TagURL.
func TestStream_applyHashtagLinks_NoTagURL(t *testing.T) {
	service := &Stream{contentService: &Content{}}

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
	service := &Stream{contentService: &Content{}}

	template := model.NewTemplate("test", nil)
	template.TagURL = "/search?q="

	stream := model.NewStream()
	stream.Content = model.NewContent()
	stream.Content.HTML = "Testing #travel here"

	service.applyHashtagLinks(&template, &stream)

	require.Equal(t, "Testing #travel here", stream.Content.HTML, "no hashtags means nothing to link")
}
