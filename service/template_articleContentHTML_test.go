package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/markdown"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Article Content HTML
 *
 * Stream.Save normalizes every Stream against its
 * template schema, and that pass is the gate for
 * content.html.  These tests run rendered article
 * HTML through the article-markdown schema (BUG-278).
 ******************************************/

// normalizeArticleContentHTML runs one content.html value through the article-markdown
// template schema, as Stream.Save does, and returns what would be stored.
func normalizeArticleContentHTML(t *testing.T, html string) string {

	t.Helper()

	template, err := loadEmbeddedTemplates(t).Load("article-markdown")
	require.NoError(t, err)

	stream := model.NewStream()
	stream.TemplateID = "article-markdown"
	stream.Content = model.Content{Format: model.ContentFormatMarkdown, HTML: html}

	_, err = template.Schema.Normalize(&stream)
	require.NoError(t, err)

	return stream.Content.HTML
}

// TestArticleContentHTML_KeepsRenderedMarkdown confirms that a save keeps what the Markdown
// renderer deliberately produced: code-block language classes, heading-link classes, and embeds.
func TestArticleContentHTML_KeepsRenderedMarkdown(t *testing.T) {

	rendered := markdown.ToHTML("# Title\n\n```go\nx := 1\n```\n\n<iframe src=\"https://example.com/video\"></iframe>\n")
	stored := normalizeArticleContentHTML(t, rendered)

	require.Contains(t, stored, `<code class="language-go">`)
	require.Contains(t, stored, `class="bi bi-link header-hover-show"`)
	require.Contains(t, stored, `<iframe src="https://example.com/video">`)
}

// TestArticleContentHTML_StillSanitizes confirms that the schema is still a gate for HTML
// that never passed through the renderer, as handler/intent_create.go can store.
func TestArticleContentHTML_StillSanitizes(t *testing.T) {

	stored := normalizeArticleContentHTML(t, `<p>Hello</p><script>alert(1)</script><img src="x" onerror="alert(2)"><a href="javascript:alert(3)">link</a>`)

	require.Contains(t, stored, "<p>Hello</p>")
	require.NotContains(t, stored, "<script")
	require.NotContains(t, stored, "onerror")
	require.NotContains(t, stored, "javascript:")
}
