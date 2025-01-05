package parse

func isEndOfToken(r rune, original string, index int) bool {

	if isHardTerminator(r) {
		return true
	}

	if isSoftTerminator(r) {
		nextRune := peekNextRune(original, index)
		return isWhitespace(nextRune)
	}

	return false
}

// isSoftTerminator returns TRUE if the provided rune matches the list of soft terminators.
// Soft Terminators (like periods) probably end a token, but only if follwed by a whitespace character.
func isSoftTerminator(r rune) bool {
	return isOneOf(r, softTerminators)
}

// isHardTerminator returns TRUE if the provided rune matches the list of hard terminators.
// Hard Terminators always end a token, regardless of what comes next.
func isHardTerminator(r rune) bool {
	return isOneOf(r, hardTerminators)
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

// peekNextRune returns the next rune in the string from the provided index.
// If the index is at the end of the string, then it returns a space character (which is a hard terminator).
func peekNextRune(value string, index int) rune {
	if index+1 < len(value) {
		return rune(value[index+1])
	}
	return ' '
}
