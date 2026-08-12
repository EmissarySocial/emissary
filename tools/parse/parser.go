package parse

import (
	"strings"
	"unicode/utf8"

	"github.com/benpate/rosetta/sliceof"
)

// Parser is an object that scans a string for matching tokens.  It can be configured using the
// `WithXXX()` optional functions to customize the scanning behavior and results.
type Parser struct {
	prefixes        []rune
	hardTerminators []rune // Runes that always end a token.  Differs per prefix -- see constants.go
	softTerminators []rune // Runes that end a token only when followed by whitespace
	remainder       *strings.Builder
	includePrefix   bool // If TRUE, include the prefix character in the final result, i.e. #hashtag or @mention (default is FALSE)
	caseSensitive   bool // If TRUE, tags keep their original case; if FALSE, tags are lower-cased (default is TRUE)
	trimRemainder   bool // If TRUE, trim leading and trailing whitespace from the remainder (default is FALSE)
}

// New returns a fully initialized Parser, with all optional parameters applied
func New(options ...Option) Parser {

	// Terminators default to the shared base set.  WithHashtagsOnly and WithMentionsOnly each
	// swap in their own -- a hashtag must break on '-' and '@' where a mention must not, so a
	// single package-wide list cannot serve both.  A mixed-prefix Parser (the default, used by
	// All) keeps the base set, and so does NOT reproduce what Hashtags+Mentions would find.
	result := Parser{
		prefixes:        []rune{'@', '#'},
		hardTerminators: baseTerminators,
		softTerminators: softTerminators,
		includePrefix:   false,
		caseSensitive:   true,
		trimRemainder:   false,
	}

	// Apply options
	result.With(options...)

	// Great success
	return result
}

func (parser *Parser) With(options ...Option) *Parser {
	for _, option := range options {
		option(parser)
	}

	return parser
}

// Parse scans the provided string, and returns a list of tags that were found, and the remainder of the string
func (parser Parser) Parse(original string) sliceof.String {

	var readyForToken = true
	var ingestingToken = false
	var currentToken strings.Builder // currentToken is the tag that we're currently ingesting

	found := sliceof.NewString() // found is the list of tags that we've found

	// Scan each rune in the original string
	for index, r := range original {

		switch {

		// If we're already ingesting a tag, then this rune is part of a tag...
		case ingestingToken:

			// If we have reached the end of a token, then collect the tag and stop ingesting
			if parser.isEndOfToken(r, original, index) {
				found = parser.foundTag(currentToken.String(), found)

				// A terminator that is ITSELF a prefix opens the next token immediately, so that
				// "#foo#bar" yields two tags.  Without this the second '#' is consumed as a plain
				// terminator and "bar" is never scanned.
				if parser.isPrefix(r) {
					parser.startToken(&currentToken, r)
					ingestingToken = true
					readyForToken = false
					continue
				}

				ingestingToken = false
				readyForToken = true
				if parser.remainder != nil {
					parser.remainder.WriteRune(r)
				}
				continue
			}

			// Otherwise, append this rune to the current tag
			currentToken.WriteRune(r)

		// If this rune is a prefix character, then start ingesting a new tag
		case readyForToken && parser.isPrefix(r):
			parser.startToken(&currentToken, r)
			ingestingToken = true
			readyForToken = false

		// Not in a tag, and not a prefix character, so just append to the remainder
		default:
			if parser.remainder != nil {
				parser.remainder.WriteRune(r)
			}

			// A tag may begin after whitespace OR after any hard terminator.  Without the second
			// half, a '#' preceded by punctuation is ignored entirely -- "(#hashtag)" and the
			// preceding-punctuation examples in FEP-eb48 ("-#hashtag", "&#hashtag") found nothing.
			readyForToken = isWhitespace(r) || parser.isHardTerminator(r)
		}
	}

	// If we were ingesting a tag when we hit the end of the original string, then add it to the final result.
	if ingestingToken {
		found = parser.foundTag(currentToken.String(), found)
	}

	// Great success
	return found
}

// isPrefix returns TRUE if the provided rune matchs the configured list of prefixes
func (parser Parser) isPrefix(r rune) bool {
	return isOneOf(r, parser.prefixes)
}

// isHardTerminator returns TRUE if the rune always ends a token for THIS Parser.
func (parser Parser) isHardTerminator(r rune) bool {
	return isOneOf(r, parser.hardTerminators)
}

// isEndOfToken returns TRUE if the rune at the provided byte offset ends the token being ingested.
func (parser Parser) isEndOfToken(r rune, original string, index int) bool {

	if parser.isHardTerminator(r) {
		return true
	}

	// A soft terminator (like a period) ends a token only at the end of a word, which is what
	// keeps the dots inside "@user@host.social" while still trimming "#hashtag."
	if isOneOf(r, parser.softTerminators) {
		return isWhitespace(peekNextRune(original, index+utf8.RuneLen(r)))
	}

	return false
}

// startToken resets the token buffer to begin a new tag at the provided prefix rune.
func (parser Parser) startToken(currentToken *strings.Builder, prefix rune) {
	currentToken.Reset()

	if parser.includePrefix {
		currentToken.WriteRune(prefix)
	}
}

func (parser Parser) foundTag(tag string, found []string) []string {
	if !parser.caseSensitive {
		tag = strings.ToLower(tag)
	}

	return append(found, tag)
}
