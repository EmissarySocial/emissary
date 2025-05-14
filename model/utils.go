package model

import (
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func objectID(value string) primitive.ObjectID {
	result, _ := primitive.ObjectIDFromHex(value)
	return result
}

// matchOne returns TRUE if the value matches one (or more) of the values in the slice
func matchOne(slice []string, value string) bool {
	for index := range slice {
		if slice[index] == value {
			return true
		}
	}

	return false
}

// matchAny returns TRUE if any of the values in slice1 are equal to any of the values in slice2
func matchAny(slice1 []string, slice2 []string) bool {

	for index1 := range slice1 {
		for index2 := range slice2 {
			if slice1[index1] == slice2[index2] {
				return true
			}
		}
	}

	return false
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
