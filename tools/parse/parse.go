package parse

import (
	"strings"

	"github.com/benpate/rosetta/sliceof"
)

// All scans for all hashtags and mentions, and returns the list of tokens found along with the remainder of the string.
func All(original string, options ...Option) (sliceof.String, string) {

	var builder strings.Builder
	parser := New(WithRemainder(&builder))
	parser.With(options...)

	tags := parser.Parse(original)
	return tags, builder.String()
}

// Hashtags scans for Hashtags only, and returns the list of tokens found.
func Hashtags(original string, options ...Option) sliceof.String {
	parser := New(WithHashtagsOnly())
	parser.With(options...)
	return parser.Parse(original)
}

// HashtagsAndRemainder scans for #Hashtags only, and returns the list of tokens found along with the remainder of the string.
func HashtagsAndRemainder(original string, options ...Option) (sliceof.String, string) {

	var builder strings.Builder
	parser := New(WithHashtagsOnly(), WithRemainder(&builder))
	parser.With(options...)

	tags := parser.Parse(original)
	return tags, builder.String()
}

// Mentions scans for @Mentions only, and returns the list of tokens found.
func Mentions(original string, options ...Option) sliceof.String {
	parser := New(WithMentionsOnly())
	parser.With(options...)
	return parser.Parse(original)
}

// MentionsAndRemainder scans for @Mentions only, and returns the list of tokens found along with the remainder of the string.
func MentionsAndRemainder(original string, options ...Option) (sliceof.String, string) {

	var builder strings.Builder
	parser := New(WithMentionsOnly(), WithRemainder(&builder))
	parser.With(options...)

	tags := parser.Parse(original)
	return tags, builder.String()
}
