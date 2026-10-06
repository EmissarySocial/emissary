package mastodon

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/toot/object"
)

const (
	articleExcerptLength = 280                // longest teaser, in characters, shown in the post and the card
	articleUntitled      = "Untitled article" // title shown for an article that has none
)

// isArticle returns TRUE for a long-form article, which the Mastodon API has no type for.
func isArticle(stream *model.Stream) bool {
	return stream.SocialRole == vocab.ObjectTypeArticle || strings.HasPrefix(stream.TemplateID, "article-")
}

// applyArticle turns a Stream's status into the closest thing a Mastodon client can show for an
// article: a short post linking to it, plus a preview card with its title, summary, image and author.
func applyArticle(status *object.Status, stream *model.Stream) {

	title := strings.TrimSpace(stream.Label)

	if title == "" {
		title = articleUntitled
	}

	teaser := articleTeaser(stream)

	// The title is the article's name, not a content warning
	status.SpoilerText = ""
	status.Sensitive = false

	// The post text is the title as a link, then the teaser
	content := `<p><a href="` + html.EscapeString(stream.URL) + `" rel="nofollow noopener noreferrer">` + html.EscapeString(title) + `</a></p>`

	if teaser != "" {
		content += `<p>` + html.EscapeString(teaser) + `</p>`
	}

	status.Content = content
	status.Card = articleCard(status, stream, title, teaser)
}

// articleTeaser returns the article's summary, or else the start of its text, as plain text.
func articleTeaser(stream *model.Stream) string {

	if summary := strings.TrimSpace(stream.Summary); summary != "" {
		return truncateText(summary, articleExcerptLength)
	}

	return truncateText(plainText(stream.Content.HTML), articleExcerptLength)
}

// articleCard builds the preview card for an article.
func articleCard(status *object.Status, stream *model.Stream, title string, teaser string) *object.PreviewCard {

	card := &object.PreviewCard{
		URL:         stream.URL,
		Title:       title,
		Description: teaser,
		Type:        "link",
		AuthorName:  stream.AttributedTo.Name,
		AuthorURL:   stream.AttributedTo.ProfileURL,
		PublishedAt: status.CreatedAt,
		Image:       articleImage(status, stream),
		Authors:     []object.PreviewCardAuthor{},
	}

	if parsed, err := url.Parse(stream.URL); err == nil && parsed.Host != "" {
		card.ProviderName = parsed.Host
		card.ProviderURL = parsed.Scheme + "://" + parsed.Host
	}

	if stream.AttributedTo.Name != "" {
		card.Authors = append(card.Authors, object.PreviewCardAuthor{Name: stream.AttributedTo.Name, URL: stream.AttributedTo.ProfileURL})
	}

	return card
}

// articleImage returns the article's cover picture: its thumbnail, else its first uploaded image.
func articleImage(status *object.Status, stream *model.Stream) string {

	if webURL(stream.IconURL) {
		return stream.IconURL
	}

	for _, media := range status.MediaAttachments {
		if media.Type == "image" && webURL(media.URL) {
			return media.URL
		}
	}

	return ""
}

// webURL returns TRUE for an absolute http or https address.
func webURL(address string) bool {

	parsed, err := url.Parse(address)

	if err != nil || parsed == nil {
		return false
	}

	return (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != ""
}

// blockTag matches the opening or closing tag of an element that starts a new line of text.
var blockTag = regexp.MustCompile(`(?i)</?(p|h[1-6]|li|ul|ol|div|br|blockquote|pre|tr|td|th)\b[^>]*>`)

// plainText strips the markup from HTML and collapses its whitespace, keeping words in
// neighbouring blocks apart.
func plainText(markup string) string {
	text := html.UnescapeString(convert.SanitizeText(blockTag.ReplaceAllString(markup, " ")))
	return strings.Join(strings.Fields(text), " ")
}

// truncateText shortens text to at most limit characters, ending with an ellipsis when it cuts.
func truncateText(text string, limit int) string {

	if utf8.RuneCountInString(text) <= limit {
		return text
	}

	runes := []rune(text)
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}
