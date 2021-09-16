package htmlconv

import "strings"

// Summary returns the first few sentences of content from an HTML document
func Summary(html string) string {

	text := ToText(html)

	// If we found a paragraph (or other break, only use the first line)
	if index := strings.Index(text, "\n"); index > -1 {
		text = text[:index]
	}

	// Remove any extra whitespace now.
	text = CollapseWhitespace(text)

	// If it's STILL too long, then truncate to 200 characters.
	if len(text) > 200 {
		text = text[:200] + "..."
	}

	return text
}
