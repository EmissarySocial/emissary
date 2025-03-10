package sorted

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnique(t *testing.T) {

	do := func(input []string, expected []string) {
		actual := Unique(input)
		require.Equal(t, expected, actual)
	}

	do([]string{}, []string{})                                // Empty values should work
	do([]string{"a"}, []string{"a"})                          // Equivalent values should work
	do([]string{"a", "a"}, []string{"a"})                     // Remove duplicate values
	do([]string{"a", "b"}, []string{"a", "b"})                // Allow different values
	do([]string{"a", "a", "b"}, []string{"a", "b"})           // Remove duplicate values
	do([]string{"a", "b", "b"}, []string{"a", "b"})           // Remove LOTS of duplicate values
	do([]string{"a", "a", "b", "b"}, []string{"a", "b"})      // AI autocomplete, whatever
	do([]string{"a", "b", "c"}, []string{"a", "b", "c"})      // AI autocomplete, whatever
	do([]string{"a", "a", "b", "c"}, []string{"a", "b", "c"}) // AI autocomplete, whatever
}
