package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestInlineSaveButton verifies that an "inline-save-button" step parses its configuration
func TestInlineSaveButton(t *testing.T) {

	step, err := NewInlineSaveButton(mapof.Any{
		"id":    "save-1",
		"class": "secondary",
		"label": "Save Now",
	})
	require.Nil(t, err)
	require.NotNil(t, step.ID)
	require.Equal(t, "secondary", step.Class)
	require.NotNil(t, step.Label)

	// Defaults: id="inline-save-button", class="primary", label="Save Changes".
	step, err = NewInlineSaveButton(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "primary", step.Class)
	require.NotNil(t, step.ID)
	require.NotNil(t, step.Label)

	require.Equal(t, "inline-save-button", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestInlineSaveButton_Form verifies that the optional "form" association is parsed
func TestInlineSaveButton_Form(t *testing.T) {

	step, err := NewInlineSaveButton(mapof.Any{"form": "edit-form"})
	require.Nil(t, err)
	require.Equal(t, "edit-form", step.Form)

	// Unset is empty, which the build step renders as no attribute at all
	step, err = NewInlineSaveButton(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "", step.Form)
}

// TestInlineSaveButton_InvalidTemplate verifies that an invalid template is rejected
func TestInlineSaveButton_InvalidTemplate(t *testing.T) {
	_, err := NewInlineSaveButton(mapof.Any{"label": "{{ .Unclosed"})
	require.NotNil(t, err)
}
