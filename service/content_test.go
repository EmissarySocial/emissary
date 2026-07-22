package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
)

// TestContent_ApplyTags confirms that a #hashtag is wrapped in a link to the given base URL.
func TestContent_ApplyTags(t *testing.T) {
	service := &Content{}

	content := model.NewContent()
	content.HTML = "Testing hashtags #travel here"

	service.ApplyTags(&content, "/search?q=", []string{"travel"})

	require.Equal(t, `Testing hashtags <a href="/search?q=%23travel" target="_blank">#travel</a> here`, content.HTML)
}

// TestContent_ApplyTags_Multiple confirms that every hashtag in the list is linked.
func TestContent_ApplyTags_Multiple(t *testing.T) {
	service := &Content{}

	content := model.NewContent()
	content.HTML = "#travel and #Food2024"

	service.ApplyTags(&content, "/search?q=", []string{"travel", "Food2024"})

	require.Equal(t, `<a href="/search?q=%23travel" target="_blank">#travel</a> and <a href="/search?q=%23Food2024" target="_blank">#Food2024</a>`, content.HTML)
}

// TestContent_ApplyTags_EmptyContent confirms that empty content is left untouched.
func TestContent_ApplyTags_EmptyContent(t *testing.T) {
	service := &Content{}

	content := model.NewContent()

	service.ApplyTags(&content, "/search?q=", []string{"travel"})

	require.Equal(t, "", content.HTML)
}

// TestContent_ApplyTags_SkipsEmptyTag confirms that an empty tag name is ignored.
func TestContent_ApplyTags_SkipsEmptyTag(t *testing.T) {
	service := &Content{}

	content := model.NewContent()
	content.HTML = "no #tag change for empty"

	service.ApplyTags(&content, "/search?q=", []string{""})

	require.Equal(t, "no #tag change for empty", content.HTML)
}

// TestContent_ApplyTags_Idempotent confirms that linkifying twice does not nest anchors.
func TestContent_ApplyTags_Idempotent(t *testing.T) {
	service := &Content{}

	content := model.NewContent()
	content.HTML = "Testing #travel here"

	service.ApplyTags(&content, "/search?q=", []string{"travel"})
	first := content.HTML

	service.ApplyTags(&content, "/search?q=", []string{"travel"})

	require.Equal(t, first, content.HTML, "second pass must not re-wrap the existing link")
}

// TestContent_ApplyTags_EscapesHostileTag confirms that a database-sourced tag name cannot break out of the
// href attribute or inject markup into the anchor text.
func TestContent_ApplyTags_EscapesHostileTag(t *testing.T) {
	service := &Content{}

	hostile := `evil"><img src=x>`

	content := model.NewContent()
	content.HTML = `before #` + hostile + ` after`

	service.ApplyTags(&content, "/search?q=", []string{hostile})

	// The raw injected markup must not survive, and the attribute must not be broken out of
	require.NotContains(t, content.HTML, "<img")
	require.NotContains(t, content.HTML, `"><`)

	// The tag must appear URL-escaped in the href and HTML-escaped in the label
	require.Contains(t, content.HTML, `%3Cimg`)
	require.Contains(t, content.HTML, `&lt;img`)
}
