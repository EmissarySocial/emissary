package htmlconv

import (
	"strings"
)

// FromText converts plain text into (lightly) formatted HTML
func FromText(text string) string {

	text = strings.Replace(text, "<", "&lt;", -1)
	text = strings.Replace(text, ">", "&gt;", -1)
	text = strings.Replace(text, `"`, "&quot;", -1)
	text = strings.Replace(text, "\n", "<br>", -1)
	text = CollapseWhitespace(text)

	return text
}
