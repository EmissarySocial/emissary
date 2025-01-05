package parse

import "strings"

type Option func(*Parser)

// WithHashtagsOnly sets the parser to only look for #Hashtags
func WithHashtagsOnly() Option {
	return func(parser *Parser) {
		parser.prefixes = []rune{'#'}
	}
}

// WithMentionsOnly sets the parser to only look for @Mentions
func WithMentionsOnly() Option {
	return func(parser *Parser) {
		parser.prefixes = []rune{'@'}
	}
}

// WithPrefixes sets the parser to look for a custom set of prefix runes.
func WithPrefixes(prefixes ...rune) Option {
	return func(parser *Parser) {
		parser.prefixes = prefixes
	}
}

// WithIncludePrefix sets the parser to include the prefix character in the final result
func WithIncludePrefix() Option {
	return func(parser *Parser) {
		parser.includePrefix = true
	}
}

// WithRemainder provides a strings.Builder to collect the remainder of the string after all tags have been found
func WithRemainder(builder *strings.Builder) Option {
	return func(parser *Parser) {
		parser.remainder = builder
	}
}

// WithCaseSensitive sets the parser to return tags in their original case (not lowercased)
func WithCaseSensitive() Option {
	return func(parser *Parser) {
		parser.caseSensitive = true
	}
}

// WithCaseInsensitive sets the parser to return tags in all lowercase
func WithCaseInsensitive() Option {
	return func(parser *Parser) {
		parser.caseSensitive = false
	}
}
