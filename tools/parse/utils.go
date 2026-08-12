package parse

import "unicode/utf8"

// isHardTerminator returns TRUE if the provided rune matches the BASE list of hard terminators.
// Only Split uses this -- the Parser carries its own per-prefix list and answers for itself.
func isHardTerminator(r rune) bool {
	return isOneOf(r, baseTerminators)
}

// isWhitespace returns TRUE if the provided rune matches the list of whitespace characters.
func isWhitespace(r rune) bool {
	return isOneOf(r, whitespace)
}

// isOneOf returns true if the provided value exists in the set
func isOneOf[T comparable](r T, set []T) bool {
	for _, s := range set {
		if r == s {
			return true
		}
	}
	return false
}

// peekNextRune returns the rune that begins at the provided BYTE offset.  If the offset is at or
// past the end of the string, then it returns a space character (which is a hard terminator).
func peekNextRune(value string, offset int) rune {

	if offset < len(value) {
		result, _ := utf8.DecodeRuneInString(value[offset:])
		return result
	}

	return ' '
}
