package service

import (
	"strings"

	"github.com/benpate/rosetta/html"
)

// textToHTML turns plain text into HTML the way Mastodon renders a post: each blank-line-separated
// paragraph in its own <p>, single line breaks as <br>, and all text escaped.
func textToHTML(text string) string {

	text = strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n")

	if text == "" {
		return ""
	}

	var builder strings.Builder

	for _, paragraph := range strings.Split(text, "\n\n") {

		if paragraph = strings.TrimSpace(paragraph); paragraph == "" {
			continue
		}

		builder.WriteString("<p>")
		builder.WriteString(html.FromText(paragraph))
		builder.WriteString("</p>")
	}

	return builder.String()
}
