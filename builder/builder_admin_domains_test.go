package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminDomains(t *testing.T) {

	// Require that Domain builder implements the PropertyFormGetter interface
	var i interface{} = Domain{}
	_, ok := i.(PropertyFormGetter)
	require.True(t, ok)
}
