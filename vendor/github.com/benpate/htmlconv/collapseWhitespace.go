package htmlconv

import (
	"regexp"
	"strings"
)

var whitespace *regexp.Regexp

func init() {
	whitespace = regexp.MustCompile(`\s+`)
}

// CollapseWhitespace converts all whitespace characters into a single SPACE character
func CollapseWhitespace(text string) string {
	result := whitespace.ReplaceAllString(text, " ")

	result = strings.TrimPrefix(result, " ")
	result = strings.TrimSuffix(result, " ")
	return result
}
