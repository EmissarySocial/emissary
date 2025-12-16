package model

import (
	"bytes"
	"strings"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

func flatten(original mapof.Object[id.Slice]) id.Slice {

	length := len(original)

	if length == 0 {
		return id.Slice{}
	}

	result := make(id.Slice, 0, length)

	for _, value := range original {
		result = append(result, value...)
	}

	return result
}

func objectID(value string) primitive.ObjectID {
	result, _ := primitive.ObjectIDFromHex(value)
	return result
}

func defaultRolesToGroupIDs(ownerID primitive.ObjectID, roleIDs ...string) Permissions {
	result := make(Permissions, 0, len(roleIDs))

	for _, roleID := range roleIDs {
		switch roleID {

		case MagicRoleAnonymous:
			result = append(result, MagicGroupIDAnonymous)

		case MagicRoleAuthenticated:
			result = append(result, MagicGroupIDAuthenticated)

		case MagicRoleMyself, MagicRoleAuthor:
			if !ownerID.IsZero() {
				result = append(result, ownerID)
			}
		}
	}

	return result
}

// markdownToHTML converts a Markdown string to HTML using the Goldmark library.
func markdownToHTML(value string, options ...goldmark.Extender) string {

	valueBytes := convert.Bytes(value)

	var buffer bytes.Buffer

	// Default options
	// https://github.com/yuin/goldmark#built-in-extensions
	if len(options) == 0 {
		options = []goldmark.Extender{
			extension.Linkify,
			extension.Typographer,
		}
	}

	md := goldmark.New(goldmark.WithExtensions(options...))

	if err := md.Convert([]byte(valueBytes), &buffer); err != nil {
		derp.Report(derp.Wrap(err, "tools.templates.functions.markdown", "Error converting Markdown to HTML"))
	}

	return buffer.String()

}

// oneOf returns TRUE if the value matches any of the provided options
func oneOf[T comparable](value T, options ...T) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

// notOneOf returns TRUE if the value does NOT match any of the provided options
func notOneOf[T comparable](value T, options ...T) bool {
	return !oneOf(value, options...)
}
