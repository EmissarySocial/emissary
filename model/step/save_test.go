package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSave verifies that a "save" step parses its configuration
func TestSave(t *testing.T) {

	step, err := NewSave(mapof.Any{
		"comment":  "Updated by {{.Author}}",
		"method":   "post",
		"on-error": []mapof.Any{{"do": "inline-error", "message": "oops"}},
	})
	require.Nil(t, err)
	require.NotNil(t, step.Comment)
	require.Equal(t, "post", step.Method)
	require.Len(t, step.OnError, 1)

	// "method" defaults to "post".
	step, err = NewSave(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "post", step.Method)

	require.Equal(t, "save", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSave_InvalidConfig verifies that an invalid config is rejected
func TestSave_InvalidConfig(t *testing.T) {
	// "method" outside the schema enum fails validation.
	_, err := NewSave(mapof.Any{"method": "not-a-method"})
	require.NotNil(t, err)
}

// TestSave_InvalidOnError verifies that an invalid on error is rejected
func TestSave_InvalidOnError(t *testing.T) {
	_, err := NewSave(mapof.Any{"on-error": []mapof.Any{{"do": "nonexistent-step"}}})
	require.NotNil(t, err)
}
