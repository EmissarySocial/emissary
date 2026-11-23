package step

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNested returns every list of steps a step holds, and nothing for a step that holds none
func TestNested(t *testing.T) {

	leaf := Include{Action: "a"}

	require.Empty(t, Nested(leaf))
	require.Empty(t, Nested(nil))

	modal := AsModal{SubSteps: []Step{leaf}, Head: []Step{leaf, leaf}}
	require.ElementsMatch(t, [][]Step{{leaf}, {leaf, leaf}}, Nested(modal))

	// An empty branch is left out
	condition := IfCondition{Then: []Step{leaf}}
	require.Equal(t, [][]Step{{leaf}}, Nested(condition))

	cases := Switch{Cases: map[string][]Step{"x": {leaf}, "y": {}}, Default: []Step{leaf, leaf}}
	require.ElementsMatch(t, [][]Step{{leaf}, {leaf, leaf}}, Nested(cases))
}

// TestIncludedActions finds an include at the top of a list and at any depth beneath it
func TestIncludedActions(t *testing.T) {

	require.Empty(t, IncludedActions(nil))
	require.Empty(t, IncludedActions([]Step{IfCondition{}}))

	steps := []Step{
		Include{Action: "first"},
		IfCondition{
			Then:      []Step{Include{Action: "second"}},
			Otherwise: []Step{Switch{Default: []Step{Include{Action: "third"}}}},
		},
		Include{Action: "first"},
	}

	require.Equal(t, []string{"first", "second", "third", "first"}, IncludedActions(steps))
}
