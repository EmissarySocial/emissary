package tests

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTypicalPatternString pins that a typical value is kept whenever it already fits the field's
// pattern, and replaced by one built from the pattern only when it does not.
func TestTypicalPatternString(t *testing.T) {

	// No pattern, or a pattern the value already fits, keeps the value
	require.Equal(t, "Sample name", typicalPatternString("Sample name", ""))
	require.Equal(t, "^[a-z]+$", typicalPatternString("^[a-z]+$", `^(\**)|(pk_(live|test)_[A-Za-z0-9]+)`))

	// A value that does not fit is replaced by one that does
	require.Equal(t, "pk_live_a", typicalPatternString("^[a-z]+$", `^pk_(live|prod|test)_\w+$`))
	require.Equal(t, "prctbl_"+strings.Repeat("a", 24), typicalPatternString("^[a-z]+$", `^prctbl_\w{24}$`))
	require.Equal(t, strings.Repeat("a", 16), typicalPatternString("sample-token", `[A-Za-z0-9._-]{16,}`))

	// A pattern that cannot compile, or cannot be built from, keeps the value for the schema to judge
	require.Equal(t, "value", typicalPatternString("value", `(`))
	require.Equal(t, "value", typicalPatternString("value", `^$`))
}

// TestPatternExample pins that every example matches its own pattern, and that a pattern whose
// shortest match is empty yields no example at all.
func TestPatternExample(t *testing.T) {

	for _, pattern := range []string{`^pk_(live|prod|test)_\w+$`, `^rk_(live|prod|test)_\w+$`, `^whsec_\w+$`, `^prctbl_\w{24}$`, `[A-Za-z0-9._-]{16,}`, `^a.b?c*d+$`, `(?i)^HELLO$`} {
		example, ok := patternExample(pattern)
		require.True(t, ok, pattern)
		require.Regexp(t, regexp.MustCompile(pattern), example, pattern)
	}

	for _, pattern := range []string{`^$`, `a*`, `(`} {
		_, ok := patternExample(pattern)
		require.False(t, ok, pattern)
	}
}
