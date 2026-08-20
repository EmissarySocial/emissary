package replace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestToLower verifies that toLower folds every rune to lowercase, whatever the input casing
func TestToLower(t *testing.T) {
	require.Equal(t, []rune("everything to lowercase"), toLower([]rune("EVERYTHING TO LOWERCASE")))
	require.Equal(t, []rune("everything to lowercase"), toLower([]rune("eVeRyTHiNG To LoWeRCaSe")))
	require.Equal(t, []rune("everything to lowercase"), toLower([]rune("EvErYtHInG tO lOwErcAsE")))
}
