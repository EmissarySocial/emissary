package replace

import "unicode"

func toLower(value []rune) []rune {
	result := make([]rune, len(value))

	for index, char := range value {
		result[index] = unicode.ToLower(char)
	}

	return result
}
