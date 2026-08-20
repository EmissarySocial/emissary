package build

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdminDomains verifies that the Domain builder satisfies the PropertyFormGetter interface
func TestAdminDomains(t *testing.T) {

	// Require that Domain builder implements the PropertyFormGetter interface
	var i interface{} = Domain{}
	_, ok := i.(PropertyFormGetter)
	require.True(t, ok)
}
