package replace

import (
	"bytes"
)

// Content replaces all occurances of a match string within the original.  It differs
// from the standard library in that: 1) it is case insensitive, 2) it does not replace
// values in HTML tags, and 3) it does not replace values inside the text of an <a> element,
// so re-running it over already-linkified content never double-wraps or nests anchors.
func Content(originalString string, matchString string, replaceString string) string {

	// RULE: An empty match string has nothing to find
	if matchString == "" {
		return originalString
	}

	var result bytes.Buffer // Final result to return to the caller

	state := stateReady
	enteringAnchor := false // TRUE while skipping an <a> open tag, so we know to enter the anchor body when it closes
	original := []rune(originalString)
	originalNoCase := toLower(original)
	matchNoCase := toLower([]rune(matchString))
	matchLength := len(matchNoCase)

	// Scan the whole original
	for index := 0; index < len(original); index++ {

		char := original[index] // nolint:scopeguard (readability)

		switch state {

		case stateReady:

			// If `char` starts an HTML tag, then switch to skipping HTML (remembering whether it opens an anchor)
			if char == '<' {
				result.WriteRune(char)
				enteringAnchor = isAnchorOpen(originalNoCase, index)
				state = stateSkipHTML
				continue
			}

			// If the match string begins here, then copy the replacement and skip past it
			if matchesAt(originalNoCase, matchNoCase, index) {
				result.WriteString(replaceString)
				index += matchLength - 1
				continue
			}

			// Fall through means no match, just write this char
			result.WriteRune(char)

		case stateSkipHTML:

			// Copy the tag verbatim
			result.WriteRune(char)

			// Keep copying until the tag closes
			if char != '>' {
				continue
			}

			// Tag closed: enter the anchor body for an <a> open tag, otherwise return to ready
			if enteringAnchor {
				enteringAnchor = false
				state = stateInsideAnchor
			} else {
				state = stateReady
			}

		case stateInsideAnchor:

			// Copy anchor content verbatim so #tags already inside a link are never re-wrapped
			result.WriteRune(char)

			// A closing </a> hands off to skipHTML, which copies the rest of the close tag and returns us to ready
			if isAnchorClose(originalNoCase, index) {
				state = stateSkipHTML
			}
		}
	}

	return result.String()
}
