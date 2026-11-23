package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseOAuthScopes verifies that every spelling of a scope string yields the same scopes
func TestParseOAuthScopes(t *testing.T) {

	check := func(value string, expected ...string) {
		t.Helper()
		t.Run(value, func(t *testing.T) {
			require.Equal(t, expected, ParseOAuthScopes(value))
		})
	}

	empty := []string{}

	check("", empty...)
	check(" ", empty...)
	check(" , ,, ", empty...)
	check("read", "read")
	check("read write", "read", "write")
	check("read  write", "read", "write")
	check(" read write ", "read", "write")
	check("read,write", "read", "write")
	check("read, write", "read", "write")
	check("read ,, write", "read", "write")
	check("read\twrite\nfollow", "read", "write", "follow")
	check("read write read", "read", "write")
	check("read:statuses write:statuses", "read:statuses", "write:statuses")
	check("Read read", "Read", "read")
}

// FuzzParseOAuthScopes confirms that no input yields an empty scope, a repeat, or a scope holding a separator
func FuzzParseOAuthScopes(f *testing.F) {

	for _, seed := range []string{"", "read write", "read,,write", " \t\n", "read read", "a,b c\td"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {

		scopes := ParseOAuthScopes(value)
		seen := map[string]bool{}

		for _, scope := range scopes {
			require.NotEmpty(t, scope)
			require.NotContains(t, scope, ",")
			require.Equal(t, []string{scope}, ParseOAuthScopes(scope))
			require.False(t, seen[scope])
			seen[scope] = true
		}

		// Parsing what was parsed changes nothing
		require.Equal(t, scopes, ParseOAuthScopes(joinScopes(scopes)))
	})
}

// joinScopes writes scopes the way a token response does
func joinScopes(scopes []string) string {

	result := ""

	for index, scope := range scopes {
		if index > 0 {
			result += " "
		}
		result += scope
	}

	return result
}
