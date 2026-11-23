package upgrades

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVersion38_ChangesNothing verifies that slot 38 succeeds without touching a database
func TestVersion38_ChangesNothing(t *testing.T) {

	// A nil database panics on first use, so success proves that nothing was read or written
	require.NoError(t, Version38(context.Background(), nil))
}
