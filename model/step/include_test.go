package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestInclude verifies that an "include" step parses its configuration
func TestInclude(t *testing.T) {
	step, err := NewInclude(mapof.Any{"action": "view"})
	require.Nil(t, err)
	require.Equal(t, "view", step.Action)
	require.Equal(t, "both", step.Method, "an include runs on every method unless told otherwise")

	require.Equal(t, "include", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestInclude_Method verifies that an "include" step reads each method it accepts, and refuses any other
func TestInclude_Method(t *testing.T) {

	for _, method := range []string{"get", "post", "both"} {
		step, err := NewInclude(mapof.Any{"action": "tag-options", "method": method})
		require.NoError(t, err)
		require.Equal(t, method, step.Method)
		require.Equal(t, "tag-options", step.Action)
	}

	_, err := NewInclude(mapof.Any{"action": "tag-options", "method": "put"})
	require.Error(t, err)
}
