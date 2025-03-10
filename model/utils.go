package model

import (
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func objectID(value string) primitive.ObjectID {
	result, _ := primitive.ObjectIDFromHex(value)
	return result
}

// ToToken returns a normalized version of the input string, stripping out
// all non-alphanumeric characters, and converting all characters to lowercase.
func ToToken(value string) string {

	var result strings.Builder

	firstCharacter := true
	specialCharacter := false

	value = strings.ToLower(value)

	for _, char := range value {
		switch char {

		case ' ', '-', '_', '.', '`', '~', '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '+', '=', '[', ']', '{', '}', '|', '\\', ';', '\'', '"', ',', '<', '>', '/', '?':
			if !firstCharacter {
				specialCharacter = true
			}

		default:

			if specialCharacter {
				result.WriteRune('-')
			}

			result.WriteRune(char)
			specialCharacter = false
			firstCharacter = false
		}
	}

	return result.String()
}
