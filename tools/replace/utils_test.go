package replace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToLower(t *testing.T) {
	require.Equal(t, []rune("everything to lowercase"), toLower([]rune("EVERYTHING TO LOWERCASE")))
	require.Equal(t, []rune("everything to lowercase"), toLower([]rune("eVeRyTHiNG To LoWeRCaSe")))
	require.Equal(t, []rune("everything to lowercase"), toLower([]rune("EvErYtHInG tO lOwErcAsE")))
}
