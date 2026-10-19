package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestEditModelObject verifies that an "edit" step parses its configuration
func TestEditModelObject(t *testing.T) {

	step, err := NewEditModelObject(mapof.Any{
		"form":    map[string]any{"type": "layout-vertical"},
		"options": []string{"delete", "{{.Token}}"},
	}, nil)
	require.Nil(t, err)
	require.Equal(t, step.Form, step.GetForm())
	require.Len(t, step.Options, 2)

	require.Equal(t, "edit", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestEditModelObject_NoForm verifies that a missing form is allowed, and yields an empty Element
func TestEditModelObject_NoForm(t *testing.T) {
	// A nil form is allowed (uses an empty Element).
	step, err := NewEditModelObject(mapof.Any{}, nil)
	require.Nil(t, err)
	require.NotNil(t, step.GetForm())
}

// TestEditModelObject_InvalidForm verifies that an invalid form is rejected
func TestEditModelObject_InvalidForm(t *testing.T) {
	_, err := NewEditModelObject(mapof.Any{"form": "not-valid-json"}, nil)
	require.NotNil(t, err)
}

// TestEditModelObject_InvalidOption verifies that an invalid option is rejected
func TestEditModelObject_InvalidOption(t *testing.T) {
	_, err := NewEditModelObject(mapof.Any{"options": []string{"{{ .Unclosed"}}, nil)
	require.NotNil(t, err)
}

// TestEditModelObject_OptionsUseFuncMap verifies that "edit" options are parsed with the
// helper functions passed in, so an option may call iif (BUG-268)
func TestEditModelObject_OptionsUseFuncMap(t *testing.T) {

	stepInfo := mapof.Any{
		"form":    map[string]any{"type": "layout-vertical"},
		"options": []string{"submit-label:{{iif .IsNew `Add a Note` `Update Note`}}"},
	}

	_, err := NewEditModelObject(stepInfo, testFuncMap())
	require.NoError(t, err)

	// Without the helper functions, the same option cannot parse
	_, err = NewEditModelObject(stepInfo, nil)
	require.Error(t, err)
}
