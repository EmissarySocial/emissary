package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestAddStream verifies that an "add-stream" step parses its configuration
func TestAddStream(t *testing.T) {

	// Explicit values are parsed through, including the with-data templates.
	step, err := NewAddStream(mapof.Any{
		"style":     "modal",
		"title":     "Create Page",
		"location":  "top",
		"state":     "draft",
		"template":  "my-template",
		"roles":     []string{"editor", "owner"},
		"with-data": mapof.Any{"label": "{{.Label}}"},
	})
	require.Nil(t, err)
	require.Equal(t, "modal", step.Style)
	require.Equal(t, "Create Page", step.Title)
	require.Equal(t, "top", step.Location)
	require.Equal(t, "draft", step.StateID)
	require.Equal(t, "my-template", step.TemplateID)
	require.Equal(t, []string{"editor", "owner"}, step.TemplateRoles)
	require.Contains(t, step.WithData, "label")

	// RequiredStates derives from StateID.
	require.Equal(t, []string{"draft"}, step.RequiredStates())
}

// TestAddStream_Defaults verifies the values an "add-stream" step falls back to when its configuration is empty
func TestAddStream_Defaults(t *testing.T) {

	step, err := NewAddStream(mapof.Any{})
	require.Nil(t, err)
	require.Equal(t, "chooser", step.Style)      // default style
	require.Equal(t, "+ Add a Page", step.Title) // default title
	require.Equal(t, "default", step.StateID)    // default state

	require.Equal(t, "add-stream", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestAddStream_InvalidLocation verifies that an invalid location is coerced to a valid value
func TestAddStream_InvalidLocation(t *testing.T) {

	// An out-of-enum location is coerced to the first allowed value ("top").
	step, err := NewAddStream(mapof.Any{"location": "not-valid"})
	require.Nil(t, err)
	require.Equal(t, "top", step.Location)
}

// TestAddStream_InvalidTemplate verifies that an invalid template is rejected
func TestAddStream_InvalidTemplate(t *testing.T) {

	// A malformed with-data template returns an error.
	_, err := NewAddStream(mapof.Any{"with-data": mapof.Any{"bad": "{{ .Unclosed"}})
	require.NotNil(t, err)

	// A malformed redirect-to template returns an error.
	_, err = NewAddStream(mapof.Any{"redirect-to": "{{ .Unclosed"})
	require.NotNil(t, err)
}
