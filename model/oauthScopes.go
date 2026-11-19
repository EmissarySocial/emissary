package model

import (
	"slices"
	"strings"
)

// ParseOAuthScopes splits a scope string into its scopes.  It is the one parser for every scope string:
// scopes may be separated by spaces, commas, or any other whitespace, in any amount, and repeats are dropped.
func ParseOAuthScopes(value string) []string {

	// Mastodon clients send commas where RFC 6749 §3.3 asks for spaces
	fields := strings.Fields(strings.ReplaceAll(value, ",", " "))
	result := make([]string, 0, len(fields))

	for _, field := range fields {
		if !slices.Contains(result, field) {
			result = append(result, field)
		}
	}

	return result
}
