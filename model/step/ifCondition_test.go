package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestIfCondition verifies that an "if" step parses its configuration
func TestIfCondition(t *testing.T) {

	step, err := NewIfCondition(mapof.Any{
		"condition": "{{.IsAuthor}}",
		"then":      []mapof.Any{{"do": "set-state", "state": "published"}},
		"else":      []mapof.Any{{"do": "set-state", "state": "draft"}},
	})
	require.Nil(t, err)
	require.NotNil(t, step.Condition)
	require.Len(t, step.Then, 1)
	require.Len(t, step.Otherwise, 1)

	// RequiredStates merges both branches.
	require.Contains(t, step.RequiredStates(), "published")
	require.Contains(t, step.RequiredStates(), "draft")

	require.Equal(t, "if", step.Name())
	require.Equal(t, "", step.RequiredModel())
}

// TestIfCondition_InvalidCondition verifies that an invalid condition is rejected
func TestIfCondition_InvalidCondition(t *testing.T) {
	_, err := NewIfCondition(mapof.Any{"condition": "{{ .Unclosed"})
	require.NotNil(t, err)
}

// TestIfCondition_InvalidThen verifies that an invalid then is rejected
func TestIfCondition_InvalidThen(t *testing.T) {
	_, err := NewIfCondition(mapof.Any{"then": []mapof.Any{{"do": "nonexistent-step"}}})
	require.NotNil(t, err)
}

// TestIfCondition_InvalidElse verifies that an invalid else is rejected
func TestIfCondition_InvalidElse(t *testing.T) {
	_, err := NewIfCondition(mapof.Any{"else": []mapof.Any{{"do": "nonexistent-step"}}})
	require.NotNil(t, err)
}
