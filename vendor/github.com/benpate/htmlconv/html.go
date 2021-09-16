package htmlconv

import (
	"fmt"
	"regexp"
	"strings"
)

var findHTML *regexp.Regexp
var spaces *regexp.Regexp
var breaks *regexp.Regexp
var paragraphs *regexp.Regexp
var divs *regexp.Regexp
var headings *regexp.Regexp
var styles *regexp.Regexp
var tags *regexp.Regexp

func init() {
	fmt.Print("init html.go... ")

	findHTML = regexp.MustCompile(`(?i)<[A-Z]+.*?>`)
	spaces = regexp.MustCompile(`[[:space:]]+`)
	breaks = regexp.MustCompile(`(?i)<br[^>]*>`)
	paragraphs = regexp.MustCompile(`(?i)<\/p>`)
	headings = regexp.MustCompile(`(?i)<\/?h[0-9][^>]*>`)
	divs = regexp.MustCompile(`(?i)<\/div>`)
	styles = regexp.MustCompile(`(?i)<style>(.*?)</style>`)
	tags = regexp.MustCompile(`<[^>]+>`)
	fmt.Println("DONE.")
}

// IsHTML returns TRUE if the string provided "looks like" HTML, in that, it has
// one or more substrings that appear to be an HTML tag
func IsHTML(html string) bool {
	return findHTML.Match([]byte(html))
}

// ToText returns a string that has been converted from HTML into plain text.
// Mostly, this means replacing block level tags (BR, P, DIV) with carriage returns.
func ToText(html string) string {

	result := html

	// Replace HTML tags
	result = spaces.ReplaceAllLiteralString(result, " ")
	result = breaks.ReplaceAllLiteralString(result, "\n")
	result = paragraphs.ReplaceAllLiteralString(result, "\n\n")
	result = headings.ReplaceAllLiteralString(result, "\n\n")
	result = divs.ReplaceAllLiteralString(result, "\n")
	result = styles.ReplaceAllLiteralString(result, "")
	result = tags.ReplaceAllLiteralString(result, "")

	// Replace HTML entities
	result = strings.Replace(result, "&#60;", "<", -1)
	result = strings.Replace(result, "&lt;", "<", -1)

	result = strings.Replace(result, "&#62;", ">", -1)
	result = strings.Replace(result, "&gt;", ">", -1)

	result = strings.Replace(result, "&#34;", `"`, -1)
	result = strings.Replace(result, "&quot;", `"`, -1)

	result = strings.Replace(result, "&#38;", "&", -1)
	result = strings.Replace(result, "&amp;", "&", -1)

	result = strings.Replace(result, "&#39;", "'", -1)
	result = strings.Replace(result, "&apos;", "'", -1)
	result = strings.Replace(result, "&apos;", "'", -1)
	result = strings.Replace(result, "&lsquo;", "'", -1)
	result = strings.Replace(result, "&rsquo;", "'", -1)

	result = strings.Replace(result, "&#124;", "|", -1)
	result = strings.Replace(result, "&#145;", "'", -1)
	result = strings.Replace(result, "&#146;", "'", -1)
	result = strings.Replace(result, "&#147;", `"`, -1)
	result = strings.Replace(result, "&#148;", `"`, -1)
	result = strings.Replace(result, "&ldquo;", `"`, -1)
	result = strings.Replace(result, "&rdquo;", `"`, -1)

	result = strings.Replace(result, "&ndash;", `-`, -1)
	result = strings.Replace(result, "&mdash;", `-`, -1)
	result = strings.Replace(result, "&#150;", `-`, -1)
	result = strings.Replace(result, "&#151;", `-`, -1)

	result = strings.Replace(result, "&#160;", " ", -1)
	result = strings.Replace(result, "&nbsp;", " ", -1)

	result = strings.Replace(result, "&#169;", "(C)", -1)
	result = strings.Replace(result, "&copy;", "(C)", -1)

	result = strings.Replace(result, "&#171;", "<<", -1)
	result = strings.Replace(result, "&laquo;", "<<", -1)

	result = strings.Replace(result, "&#187;", ">>", -1)
	result = strings.Replace(result, "&raquo;", ">>", -1)

	result = strings.Replace(result, "&#174;", "(R)", -1)
	result = strings.Replace(result, "&reg;", "(R)", -1)

	result = strings.Replace(result, "&#8230;", "...", -1)
	result = strings.Replace(result, "&hellip;", "...", -1)

	result = strings.Replace(result, "&#8249;", "<", -1)
	result = strings.Replace(result, "&lsaquo;", "<", -1)

	result = strings.Replace(result, "&#8250;", ">", -1)
	result = strings.Replace(result, "&rsaquo;", "<", -1)

	result = strings.Trim(result, " ")

	return result
}

// RemoveTags removes all HTML tags from a string, returning plain text without any formatting.
func RemoveTags(html string) string {
	return tags.ReplaceAllLiteralString(html, "")
}
