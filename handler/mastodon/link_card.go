package mastodon

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"golang.org/x/net/html"
)

// linkCardLookupTimeout bounds the cache read, so a slow database never holds up a timeline
const linkCardLookupTimeout = 2 * time.Second

// applyLinkCard fills a post's preview card from the cached metadata of its first link,
// and queues that page to be read when it is not cached yet, so the card appears on a later load
func applyLinkCard(factory *service.Factory, session data.Session, status *object.Status) {

	const location = "handler.mastodon.applyLinkCard"

	// A post that has a card already (an article) or shows media gets none
	if status.Card != nil || len(status.MediaAttachments) > 0 {
		return
	}

	link := firstCardLink(status)

	if link == "" {
		return
	}

	ctx, cancel := context.WithTimeout(session.Context(), linkCardLookupTimeout)
	defer cancel()

	linkPreviewService := factory.LinkPreview()
	preview, err := linkPreviewService.LoadByURL(ctx, link)

	// Not read yet, or expired: ask the queue, and show no card this time
	if derp.IsNotFound(err) {
		linkPreviewService.QueueFetch(session, link)
		return
	}

	// RULE: A card is an extra, so a database failure shows the post without one
	if err != nil {
		derp.Report(derp.Wrap(err, location, "Loading link preview", link))
		return
	}

	status.Card = preview.Card()
}

// firstCardLink returns the first link in a post's text that points to a web page,
// skipping mentions, hashtags, and links back to the post itself, as Mastodon does
func firstCardLink(status *object.Status) string {

	if status.Content == "" {
		return ""
	}

	tokenizer := html.NewTokenizer(strings.NewReader(status.Content))

	// The anchor being read: its address, whether its markup rules it out, and its text so far
	href := ""
	skip := false
	insideAnchor := false
	var text strings.Builder

	for {

		switch tokenizer.Next() {

		case html.ErrorToken:
			return ""

		case html.StartTagToken:

			name, hasAttributes := tokenizer.TagName()

			if string(name) != "a" {
				continue
			}

			insideAnchor = true
			href, skip = "", false
			text.Reset()

			for hasAttributes {

				var key, value []byte
				key, value, hasAttributes = tokenizer.TagAttr()

				switch string(key) {

				case "href":
					href = strings.TrimSpace(string(value))

				case "class":
					classes := strings.Fields(string(value))
					skip = skip || slices.Contains(classes, "mention") || slices.Contains(classes, "hashtag")

				case "rel":
					skip = skip || slices.Contains(strings.Fields(string(value)), "tag")
				}
			}

		case html.TextToken:

			if insideAnchor {
				text.Write(tokenizer.Text())
			}

		case html.EndTagToken:

			name, _ := tokenizer.TagName()

			if string(name) != "a" || !insideAnchor {
				continue
			}

			insideAnchor = false

			if isCardLink(status, href, skip, text.String()) {
				return href
			}
		}
	}
}

// isCardLink returns TRUE if an anchor in a post can become its preview card
func isCardLink(status *object.Status, href string, skip bool, text string) bool {

	if skip || !service.IsLinkPreviewURL(href) {
		return false
	}

	// Servers that leave mentions and hashtags unmarked still start their text with @ or #
	if trimmed := strings.TrimSpace(text); strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "#") {
		return false
	}

	// A link back to the post itself describes nothing new
	return href != status.URL && href != status.URI
}
