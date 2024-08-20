package camper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalCapitalization(t *testing.T) {

	require.Equal(t, "Like", CanonicalCapitalization("like"))
	require.Equal(t, "Like", CanonicalCapitalization("LIKE"))
	require.Equal(t, "Like", CanonicalCapitalization("Like"))
	require.Equal(t, "Like", CanonicalCapitalization("lIkE"))
}
