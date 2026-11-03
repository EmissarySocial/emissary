package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSort verifies that a "sort" step parses its configuration
func TestSort(t *testing.T) {

	step, err := NewSort(mapof.Any{"model": "Stream", "keys": "token", "values": "order", "direction": "desc", "message": "Reordered"})
	require.Nil(t, err)
	require.Equal(t, "Stream", step.Model)
	require.Equal(t, "token", step.Keys)
	require.Equal(t, "order", step.Values)
	require.Equal(t, "desc", step.Direction)
	require.Equal(t, "Reordered", step.Message)

	// Defaults: keys "_id", values "rank", direction "asc".
	step, err = NewSort(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "_id", step.Keys)
	require.Equal(t, "rank", step.Values)
	require.Equal(t, "asc", step.Direction)

	require.Equal(t, "sort", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSort_Direction verifies which "direction" values a "sort" step accepts
func TestSort_Direction(t *testing.T) {

	accepts := func(value string, expected string) {
		t.Helper()
		step, err := NewSort(mapof.Any{"direction": value})
		require.Nil(t, err, "direction %q", value)
		require.Equal(t, expected, step.Direction, "direction %q", value)
	}

	rejects := func(value string) {
		t.Helper()
		_, err := NewSort(mapof.Any{"direction": value})
		require.NotNil(t, err, "direction %q", value)
	}

	accepts("", "asc")
	accepts("asc", "asc")
	accepts("desc", "desc")
	accepts("ASC", "asc")
	accepts("Desc", "desc")

	rejects("descending")
	rejects("up")
	rejects(" desc")
	rejects("asc,desc")
	rejects("\xff")
}
