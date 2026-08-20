package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestScheduleDelete verifies that a "schedule-delete" step parses its configuration
func TestScheduleDelete(t *testing.T) {

	step, err := NewScheduleDelete(mapof.Any{
		"days":    "7",
		"hours":   "{{.Hours}}",
		"minutes": "30",
		"seconds": "0",
	})
	require.Nil(t, err)
	require.NotNil(t, step.Days)
	require.NotNil(t, step.Hours)
	require.NotNil(t, step.Minutes)
	require.NotNil(t, step.Seconds)

	require.Equal(t, "schedule-delete", step.Name())
	require.Equal(t, "Stream", step.RequiredModel())
	require.Equal(t, []string{}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestScheduleDelete_InvalidTemplate verifies that an invalid template is rejected
func TestScheduleDelete_InvalidTemplate(t *testing.T) {
	_, err := NewScheduleDelete(mapof.Any{"days": "{{ .Unclosed"})
	require.NotNil(t, err)
}
