package mention

import (
	"strings"

	"github.com/benpate/rosetta/sliceof"
)

type Scanner struct {
	prefixes      []rune // List of prefixes to scan for (commonly # and/or @)
	terminators   []rune // List of terminators to scan for (commonly space, period, comma, etc)
	includePrefix bool   // If TRUE, include the prefix character in the final result, i.e. #hashtag or @mention (default is FALSE)
	caseSensitive bool   // If TRUE, tags are case sensitive (default is FALSE)
}

// New returns a fully initialized Scanner, with all optional parameters applied
func New(options ...Option) Scanner {

	result := Scanner{
		prefixes:      []rune{'#'},
		terminators:   []rune{' ', '.', ',', '!', '?', ';', ':', '%', '(', ')', '\t', '\n'},
		includePrefix: false,
		caseSensitive: false,
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

func (scanner *Scanner) With(options ...Option) *Scanner {
	for _, option := range options {
		option(scanner)
	}

	return scanner
}

// Parse scans the provided string, and returns a list of tags that were found, and the remainder of the string
func (scanner Scanner) Parse(original string) (sliceof.String, string) {

	var ingestingTag = false       // ingestingTag is TRUE when we're currently ingesting a tag
	var remainder strings.Builder  // remainder is the text that remains after all tags have been removed
	var currentTag strings.Builder // currentTag is the tag that we're currently ingesting
	found := sliceof.NewString()   // found is the list of tags that we've found

	// Scan each run in the original string
	for _, r := range original {

		switch {

		// If we're already ingesting a tag, then this rune is part of a tag...
		case ingestingTag:

			// If it's a terminators, then collect the tag and stop ingesting
			if scanner.isTerminator(r) {
				found = scanner.foundTag(currentTag.String(), found)
				remainder.WriteRune(r)
				ingestingTag = false
				continue
			}

			// Otherwise, append this rune to the current tag
			currentTag.WriteRune(r)

		// If this rune is a prefix character, then start ingesting a new tag
		case scanner.isPrefix(r):
			currentTag.Reset()
			ingestingTag = true

			if scanner.includePrefix {
				currentTag.WriteRune(r)
			}

		// Not in a tag, and not a prefix character, so just append to the remainder
		default:
			remainder.WriteRune(r)
		}
	}

	// If we were ingesting a tag when we hit the end of the original string, then add it to the final result.
	if ingestingTag {
		found = scanner.foundTag(currentTag.String(), found)
	}

	// Great success
	return found, remainder.String()
}

// ParseTagsOnly scans the provided string, and returns a list of tags that were found
func (scanner Scanner) ParseTagsOnly(original string) sliceof.String {

	var ingestingTag = false       // ingestingTag is TRUE when we're currently ingesting a tag
	var currentTag strings.Builder // currentTag is the tag that we're currently ingesting
	found := sliceof.NewString()   // found is the list of tags that we've found

	// Scan each run in the original string
	for _, r := range original {

		switch {

		// If we're already ingesting a tag, then this rune is part of a tag...
		case ingestingTag:

			// If it's a terminators, then collect the tag and stop ingesting
			if scanner.isTerminator(r) {
				found = scanner.foundTag(currentTag.String(), found)
				ingestingTag = false
				continue
			}

			// Otherwise, append this rune to the current tag
			currentTag.WriteRune(r)

		// If this rune is a prefix character, then start ingesting a new tag
		case scanner.isPrefix(r):
			currentTag.Reset()
			ingestingTag = true

			if scanner.includePrefix {
				currentTag.WriteRune(r)
			}

		}
	}

	// If we were ingesting a tag when we hit the end of the original string, then add it to the final result.
	if ingestingTag {
		found = scanner.foundTag(currentTag.String(), found)
	}

	// Great success
	return found
}

// isPrefix returns TRUE if the provided rune matchs the list of prefixes
func (scanner Scanner) isPrefix(r rune) bool {
	for _, prefix := range scanner.prefixes {
		if r == prefix {
			return true
		}
	}
	return false
}

// isTerminator returns TRUE if the provided rune matches the list of terminators
func (scanner Scanner) isTerminator(r rune) bool {
	for _, terminator := range scanner.terminators {
		if r == terminator {
			return true
		}
	}
	return false
}

func (scanner Scanner) foundTag(tag string, found []string) []string {

	if !scanner.caseSensitive {
		tag = strings.ToLower(tag)
	}

	return append(found, tag)
}
