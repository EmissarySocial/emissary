package build

import (
	"io"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// switchTestBuilder is a Builder whose only data is the value that a switch expression reads
type switchTestBuilder struct {
	Builder
	layout string
}

// Layout returns the value that the test expressions switch on
func (b switchTestBuilder) Layout() string { return b.layout }

// factory implements the Builder interface; no step under test reaches the factory
func (b switchTestBuilder) factory() Factory { return unusedFactory{} }

// switchTestStep parses a "switch" step from its configuration, failing the test if it is invalid
func switchTestStep(t *testing.T, stepInfo mapof.Any) step.Switch {
	t.Helper()

	parsed, err := step.NewSwitch(stepInfo, nil)
	require.Nil(t, err)
	return parsed
}

// runSwitch runs one parsed step through a Pipeline, which also proves its ExecutableStep case
func runSwitch(parsed step.Switch, layout string, method ActionMethod) PipelineResult {
	return Pipeline([]step.Step{parsed}).Execute(nil, switchTestBuilder{layout: layout}, io.Discard, method)
}

// switchTestEvent returns a trigger-event step, which records which branch ran in a POST's events
func switchTestEvent(value string) map[string]any {
	return map[string]any{"do": "trigger-event", "event": "ran", "value": value}
}

// TestStepSwitch_Post verifies that a POST runs exactly the matching case, or the default
func TestStepSwitch_Post(t *testing.T) {

	parsed := switchTestStep(t, mapof.Any{
		"expression": "{{.Layout}}",
		"cases": map[string]any{
			"A": []any{switchTestEvent("A")},
			"B": switchTestEvent("B"),
			"":  switchTestEvent("empty"),
		},
		"default": switchTestEvent("default"),
	})

	tests := []struct {
		layout   string
		expected string
	}{
		{layout: "A", expected: "A"},
		{layout: "B", expected: "B"},
		{layout: "", expected: "empty"},
		{layout: "  A\n", expected: "A"},
		{layout: "C", expected: "default"},
		{layout: "a", expected: "default"},
		{layout: "AB", expected: "default"},
	}

	for _, test := range tests {
		result := runSwitch(parsed, test.layout, ActionMethodPost)
		require.Nil(t, result.Error, "layout %q", test.layout)
		require.False(t, result.Halt, "layout %q", test.layout)
		require.Equal(t, mapof.String{"ran": test.expected}, result.Events, "layout %q", test.layout)
	}
}

// TestStepSwitch_Get verifies that a GET runs the matching case in the GET phase
func TestStepSwitch_Get(t *testing.T) {

	parsed := switchTestStep(t, mapof.Any{
		"expression": "{{.Layout}}",
		"cases":      map[string]any{"A": map[string]any{"do": "halt"}},
	})

	require.True(t, runSwitch(parsed, "A", ActionMethodGet).Halt, "case A halts")
	require.False(t, runSwitch(parsed, "B", ActionMethodGet).Halt, "no case and no default does nothing")
}

// TestStepSwitch_NoMatchNoDefault verifies that an unmatched value with no default continues untouched
func TestStepSwitch_NoMatchNoDefault(t *testing.T) {

	parsed := switchTestStep(t, mapof.Any{
		"expression": "{{.Layout}}",
		"cases":      map[string]any{"A": switchTestEvent("A")},
	})

	result := runSwitch(parsed, "Z", ActionMethodPost)
	require.Nil(t, result.Error)
	require.False(t, result.Halt)
	require.Empty(t, result.Events)
}

// TestStepSwitch_EmptyCase verifies that a matching case with no steps does not fall through to the default
func TestStepSwitch_EmptyCase(t *testing.T) {

	parsed := switchTestStep(t, mapof.Any{
		"expression": "{{.Layout}}",
		"cases":      map[string]any{"A": []any{}},
		"default":    switchTestEvent("default"),
	})

	result := runSwitch(parsed, "A", ActionMethodPost)
	require.Nil(t, result.Error)
	require.Empty(t, result.Events)
}

// TestStepSwitch_HaltStopsTheCase verifies that a halt inside a case stops that case and the pipeline
func TestStepSwitch_HaltStopsTheCase(t *testing.T) {

	parsed := switchTestStep(t, mapof.Any{
		"expression": "{{.Layout}}",
		"cases": map[string]any{
			"A": []any{switchTestEvent("before"), map[string]any{"do": "halt"}, switchTestEvent("after")},
		},
	})

	result := runSwitch(parsed, "A", ActionMethodPost)
	require.True(t, result.Halt)
	require.Equal(t, mapof.String{"ran": "before"}, result.Events)
}

// TestStepSwitch_ExpressionError verifies that an expression that cannot render halts with an error
func TestStepSwitch_ExpressionError(t *testing.T) {

	parsed := switchTestStep(t, mapof.Any{
		"expression": "{{.Missing}}",
		"cases":      map[string]any{"A": switchTestEvent("A")},
		"default":    switchTestEvent("default"),
	})

	for _, method := range []ActionMethod{ActionMethodGet, ActionMethodPost} {
		result := runSwitch(parsed, "A", method)
		require.True(t, result.Halt)
		require.Error(t, result.Error)
		require.Empty(t, result.Events, "no branch runs when the expression fails")
	}
}

// TestStepSwitch_NestedError verifies that an error inside a case reaches the outer pipeline
func TestStepSwitch_NestedError(t *testing.T) {

	inner := map[string]any{
		"do":         "switch",
		"expression": "{{.Missing}}",
		"cases":      map[string]any{"A": switchTestEvent("inner")},
	}

	parsed := switchTestStep(t, mapof.Any{
		"expression": "{{.Layout}}",
		"cases":      map[string]any{"A": inner},
		"default":    inner,
	})

	for _, layout := range []string{"A", "Z"} {
		result := runSwitch(parsed, layout, ActionMethodPost)
		require.True(t, result.Halt, "layout %q", layout)
		require.Error(t, result.Error, "layout %q", layout)
	}
}

// TestStepSwitch_SharedAcrossRequests verifies that concurrent requests each run their own case,
// and leave the shared, parsed step unchanged
func TestStepSwitch_SharedAcrossRequests(t *testing.T) {

	parsed := switchTestStep(t, mapof.Any{
		"expression": "{{.Layout}}",
		"cases": map[string]any{
			"A": switchTestEvent("A"),
			"B": switchTestEvent("B"),
		},
		"default": switchTestEvent("default"),
	})

	caseCount := len(parsed.Cases)
	defaultCount := len(parsed.Default)

	// Collect each request's events, keyed by the layout it sent
	layouts := []string{"A", "B", "Z", "A", "B", "Z", "A", "B"}
	results := make([]mapof.String, len(layouts))

	var wait sync.WaitGroup
	for index, layout := range layouts {
		wait.Go(func() {
			results[index] = runSwitch(parsed, layout, ActionMethodPost).Events
		})
	}
	wait.Wait()

	expected := map[string]string{"A": "A", "B": "B", "Z": "default"}
	for index, layout := range layouts {
		require.Equal(t, mapof.String{"ran": expected[layout]}, results[index], "request %d (%q)", index, layout)
	}

	require.Len(t, parsed.Cases, caseCount)
	require.Len(t, parsed.Default, defaultCount)
}
