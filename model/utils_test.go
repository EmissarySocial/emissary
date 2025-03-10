package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToToken(t *testing.T) {

	do := func(input, expected string) {
		require.Equal(t, ToToken(input), expected)
	}

	do("!!!! Ignore leading chars", "ignore-leading-chars")   // Ignore leading special characters
	do("Ignore trailing chars !!!!", "ignore-trailing-chars") // Ignore trailing special characters
	do("Hello, World!", "hello-world")                        // Lowercase, and replace special characters with "-"
	do("Hägen Däs", "hägen-däs")                              // Allow diacritics
	do("Æthelflad", "æthelflad")                              // Æthenflad is a bad-ass.
	do("category:value", "category:value")                    // Intentionally allowing ":" because it's used for tag categories
}
