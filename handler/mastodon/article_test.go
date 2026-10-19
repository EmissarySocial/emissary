package mastodon

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// newTestArticle returns a published Markdown article with a title, summary, thumbnail, author and body.
func newTestArticle() model.Stream {

	stream := model.NewStream()
	stream.TemplateID = "article-markdown"
	stream.SocialRole = "Article"
	stream.URL = "https://example.com/6aa28844953fd4bd9bb5638e"
	stream.Label = "Why <Emissary> & friends"
	stream.Summary = "A short tour of the toolkit."
	stream.IconURL = "https://cdn.example.com/cover.png"
	stream.AttributedTo = model.PersonLink{Name: "Ben Pate", ProfileURL: "https://example.com/@ben"}
	stream.Content = model.NewHTMLContent("<h1>Why</h1><p>First paragraph with <b>bold</b> text &amp; more.</p>")

	return stream
}

// TestIsArticle covers both ways a stream is recognised as an article, and a short post that is not.
func TestIsArticle(t *testing.T) {

	article := newTestArticle()
	require.True(t, isArticle(&article))

	byTemplate := model.NewStream()
	byTemplate.TemplateID = "article-html"
	require.True(t, isArticle(&byTemplate))

	byRole := model.NewStream()
	byRole.SocialRole = "Article"
	require.True(t, isArticle(&byRole))

	post := model.NewStream()
	post.TemplateID = "outbox-message"
	post.SocialRole = "Note"
	require.False(t, isArticle(&post))
}

// TestApplyArticle_BuildsAPostAndACard checks the text, the escaping of the title, and every card field.
func TestApplyArticle_BuildsAPostAndACard(t *testing.T) {

	stream := newTestArticle()
	status := stream.Toot()

	require.Equal(t, "Why <Emissary> & friends", status.SpoilerText, "the plain conversion treats a label as a content warning")

	applyArticle(&status, &stream)

	require.Equal(t, "", status.SpoilerText)
	require.False(t, status.Sensitive)
	require.Equal(t, `<p><a href="https://example.com/6aa28844953fd4bd9bb5638e" rel="nofollow noopener noreferrer">Why &lt;Emissary&gt; &amp; friends</a></p><p>A short tour of the toolkit.</p>`, status.Content)

	card := status.Card
	require.NotNil(t, card)
	require.Equal(t, "https://example.com/6aa28844953fd4bd9bb5638e", card.URL)
	require.Equal(t, "Why <Emissary> & friends", card.Title, "the card holds plain text, not markup")
	require.Equal(t, "A short tour of the toolkit.", card.Description)
	require.Equal(t, "link", card.Type)
	require.Equal(t, "Ben Pate", card.AuthorName)
	require.Equal(t, "https://example.com/@ben", card.AuthorURL)
	require.Equal(t, "example.com", card.ProviderName)
	require.Equal(t, "https://example.com", card.ProviderURL)
	require.Equal(t, "https://cdn.example.com/cover.png", card.Image)
	require.Equal(t, status.CreatedAt, card.PublishedAt)
	require.Len(t, card.Authors, 1)
}

// TestApplyArticle_FallsBackWhenThePiecesAreMissing covers an article with no title, summary, thumbnail or author.
func TestApplyArticle_FallsBackWhenThePiecesAreMissing(t *testing.T) {

	stream := newTestArticle()
	stream.Label = ""
	stream.Summary = ""
	stream.IconURL = "not a web address"
	stream.AttributedTo = model.PersonLink{}

	status := stream.Toot()
	status.MediaAttachments = []object.MediaAttachment{
		{Type: "video", URL: "https://cdn.example.com/clip.mp4"},
		{Type: "image", URL: "https://cdn.example.com/first.png"},
	}

	applyArticle(&status, &stream)

	require.Equal(t, "Untitled article", status.Card.Title)
	require.Equal(t, "Why First paragraph with bold text & more.", status.Card.Description, "the teaser is the start of the body, as plain text")
	require.Equal(t, "https://cdn.example.com/first.png", status.Card.Image, "the first uploaded image stands in for a missing thumbnail")
	require.Empty(t, status.Card.Authors)
}

// TestArticleTeaser_ShortensLongText confirms a long body is cut with an ellipsis, counting characters not bytes.
func TestArticleTeaser_ShortensLongText(t *testing.T) {

	stream := newTestArticle()
	stream.Summary = ""
	stream.Content = model.NewHTMLContent("<p>" + strings.Repeat("é", 500) + "</p>")

	teaser := articleTeaser(&stream)

	require.Equal(t, articleExcerptLength, len([]rune(teaser)))
	require.True(t, strings.HasSuffix(teaser, "…"))
}
