package parse

import (
	"strings"

	"github.com/benpate/rosetta/sliceof"
)

// Parser is an object that scans a string for matching tokens.  It can be configured using the
// `WithXXX()` optional functions to customize the scanning behavior and results.
type Parser struct {
	prefixes      []rune
	remainder     *strings.Builder
	includePrefix bool // If TRUE, include the prefix character in the final result, i.e. #hashtag or @mention (default is FALSE)
	caseSensitive bool // If TRUE, tags are case sensitive (default is FALSE)
}

// New returns a fully initialized Parser, with all optional parameters applied
func New(options ...Option) Parser {

	// Set up defaults
	result := Parser{
		prefixes:      []rune{'@', '#'},
		includePrefix: false,
		caseSensitive: true,
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

	var ingestingToken bool
	var currentToken strings.Builder // currentToken is the tag that we're currently ingesting

	found := sliceof.NewString() // found is the list of tags that we've found

	// Scan each run in the original string
	for index, r := range original {

		switch {

		// If we're already ingesting a tag, then this rune is part of a tag...
		case ingestingToken:

			// If we have reached the end of a token, then collect the tag and stop ingesting
			if isEndOfToken(r, original, index) {
				found = parser.foundTag(currentToken.String(), found)
				if parser.remainder != nil {
					parser.remainder.WriteRune(r)
				}
				ingestingToken = false
				continue
			}

			// Otherwise, append this rune to the current tag
			currentToken.WriteRune(r)

		// If this rune is a prefix character, then start ingesting a new tag
		case parser.isPrefix(r):
			currentToken.Reset()
			ingestingToken = true

			if parser.includePrefix {
				currentToken.WriteRune(r)
			}

		// Not in a tag, and not a prefix character, so just append to the remainder
		default:
			if parser.remainder != nil {
				parser.remainder.WriteRune(r)
			}
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

func (parser Parser) foundTag(tag string, found []string) []string {
	if !parser.caseSensitive {
		tag = strings.ToLower(tag)
	}

	return append(found, tag)
}
