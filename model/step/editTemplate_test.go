package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestEditTemplate verifies that an "edit-template" step parses its configuration
func TestEditTemplate(t *testing.T) {

	step, err := NewEditTemplate(mapof.Any{
		"title":          "Choose Template",
		"templateId":     true,
		"inboxTemplate":  true,
		"outboxTemplate": false, // false values are not added to Paths
	})
	require.Nil(t, err)
	require.Equal(t, "Choose Template", step.Title)
	require.Contains(t, step.Paths, "templateId")
	require.Contains(t, step.Paths, "inboxTemplate")
	require.NotContains(t, step.Paths, "outboxTemplate")

	require.Equal(t, "edit-template", step.Name())
	require.Equal(t, "", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestEditTemplate_InvalidKey verifies that an invalid key is rejected
func TestEditTemplate_InvalidKey(t *testing.T) {
	// Any key other than do/title/templateId/inboxTemplate/outboxTemplate is rejected.
	_, err := NewEditTemplate(mapof.Any{"unexpectedKey": true})
	require.NotNil(t, err)
}
