package replace

import (
	"bytes"
	"unicode"

	"github.com/benpate/rosetta/slice"
)

// Content replaces all occurances of a match string within the original.  It differs
// from the standard library in that: 1) it is case insensitive, 2) it does not replace
// values in HTML tags.
func Content(originalString string, matchString string, replaceString string) string {

	var result bytes.Buffer // Final result to return to the caller
	var state int = stateReady

	original := []rune(originalString)
	originalNoCase := toLower(original)
	matchNoCase := toLower([]rune(matchString))
	matchLength := len(matchNoCase)

	firstMatchRune := unicode.ToLower(matchNoCase[0])

	// Copy from original as lowercase runes
	for index, char := range original {
		originalNoCase[index] = unicode.ToLower(char)
	}

	// Scan the whole original
	for index := 0; index < len(original); index++ {

		char := original[index] // nolint:scopeguard (readability)

		switch state {

		case stateReady:

			// If `char` starts an HTML tag, then switch to skipping HTML
			if char == '<' {
				result.WriteRune(char)
				state = stateSkipHTML
				continue
			}

			// Bounds check
			if index+matchLength <= len(original) {

				// If `char` is a match then lets search for the whole string
				if unicode.ToLower(char) == firstMatchRune {

					// If the next few characters equal the `match` value, then copy
					// the `replace` value instead, and increment the counter
					if slice.Equal(originalNoCase[index:index+matchLength], matchNoCase) {
						result.WriteString(replaceString)
						index += matchLength - 1
						continue
					}
				}
			}

			// Fall through means no match, just write this char
			result.WriteRune(char)

		case stateSkipHTML:

			// All `char` values write to the result
			result.WriteRune(char)

			// If this `char` ends an HTML tag, then return to ready state
			if char == '>' {
				state = stateReady
			}
		}
	}

	return result.String()
}
