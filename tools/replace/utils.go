package replace

import (
	"unicode"

	"github.com/benpate/rosetta/slice"
)

// toLower returns a copy of the rune slice with every rune folded to lower case.
func toLower(value []rune) []rune {
	result := make([]rune, len(value))

	for index, char := range value {
		result[index] = unicode.ToLower(char)
	}

	return result
}

// matchesAt returns TRUE if the `match` runes occur in `runes` starting at `index`.
func matchesAt(runes []rune, match []rune, index int) bool {

	if index+len(match) > len(runes) {
		return false
	}

	return slice.Equal(runes[index:index+len(match)], match)
}

// isSpace returns TRUE if the rune is an HTML whitespace character.
func isSpace(char rune) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

// isAnchorOpen returns TRUE if an <a> opening tag begins at `index` in the (already lower-cased) runes.
// It matches "<a" only when followed by whitespace or ">", so tags like <article> are not mistaken for anchors.
func isAnchorOpen(runes []rune, index int) bool {

	// Need at least "<a" plus a name-terminating character
	if index+2 >= len(runes) {
		return false
	}

	if runes[index] != '<' || runes[index+1] != 'a' {
		return false
	}

	return runes[index+2] == '>' || isSpace(runes[index+2])
}

// isAnchorClose returns TRUE if a </a> closing tag begins at `index` in the (already lower-cased) runes.
func isAnchorClose(runes []rune, index int) bool {

	// Need at least "</a" plus a name-terminating character
	if index+3 >= len(runes) {
		return false
	}

	if runes[index] != '<' || runes[index+1] != '/' || runes[index+2] != 'a' {
		return false
	}

	return runes[index+3] == '>' || isSpace(runes[index+3])
}
