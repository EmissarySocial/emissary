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
	value = strings.ToLower(value)
	for _, char := range value {
		switch char {
		case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
			result.WriteRune(char)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			result.WriteRune(char)
		case ':':
			result.WriteRune(':')
		}
	}

	return result.String()
}
