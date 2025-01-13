package parse

import (
	"strings"

	"github.com/benpate/rosetta/sliceof"
)

// Split returns all tokens in a string, separated by any whitespace, and omitting prefixes.
func Split(original string) sliceof.String {

	var current strings.Builder
	result := make([]string, 0)

	for _, char := range original {

		// Skip prefixes
		if char == '#' {
			continue
		}

		// Split on whitespace
		if isHardTerminator(char) {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			continue
		}

		// Otherwise, collect the character for the next token
		current.WriteRune(char)
	}

	// One last check, in case there's a token at the end of a string
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	// Return all tokens
	return result
}
