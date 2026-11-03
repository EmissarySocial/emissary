package step

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestSwitch verifies that a "switch" step parses its expression, every case, and its default
func TestSwitch(t *testing.T) {

	step, err := NewSwitch(mapof.Any{
		"expression": "{{.Data `layout`}}",
		"cases": map[string]any{
			"A": []any{
				map[string]any{"do": "set-state", "state": "alpha"},
				map[string]any{"do": "halt"},
			},
			"B": map[string]any{"do": "set-state", "state": "beta"},
		},
		"default": []any{map[string]any{"do": "set-state", "state": "other"}},
	}, nil)

	require.Nil(t, err)
	require.NotNil(t, step.Expression)
	require.Equal(t, "switch", step.Name())
	require.Equal(t, "", step.RequiredModel())

	// A list of steps keeps its order, and a single step is a pipeline of one
	require.Len(t, step.Cases, 2)
	require.Len(t, step.Cases["A"], 2)
	require.Equal(t, "set-state", step.Cases["A"][0].Name())
	require.Equal(t, "halt", step.Cases["A"][1].Name())
	require.Len(t, step.Cases["B"], 1)
	require.Equal(t, "set-state", step.Cases["B"][0].Name())
	require.Len(t, step.Default, 1)

	// RequiredStates merges the default and every case, in key order
	require.Equal(t, []string{"other", "alpha", "beta"}, step.RequiredStates())
	require.Equal(t, []string{}, step.RequiredRoles())
}

// TestSwitch_MapOfAny verifies that cases and steps given as mapof.Any parse the same as plain maps
func TestSwitch_MapOfAny(t *testing.T) {

	step, err := NewSwitch(mapof.Any{
		"expression": "{{.Label}}",
		"cases": mapof.Any{
			"A": mapof.Any{"do": "halt"},
			"B": []mapof.Any{{"do": "halt"}, {"do": "halt"}},
		},
		"default": mapof.Any{"do": "halt"},
	}, nil)

	require.Nil(t, err)
	require.Len(t, step.Cases["A"], 1)
	require.Len(t, step.Cases["B"], 2)
	require.Len(t, step.Default, 1)
}

// TestSwitch_NoDefault verifies that the default is optional, and empty when omitted or null
func TestSwitch_NoDefault(t *testing.T) {

	for _, stepInfo := range []mapof.Any{
		{"expression": "{{.Label}}", "cases": map[string]any{"A": map[string]any{"do": "halt"}}},
		{"expression": "{{.Label}}", "cases": map[string]any{"A": map[string]any{"do": "halt"}}, "default": nil},
	} {
		step, err := NewSwitch(stepInfo, nil)
		require.Nil(t, err)
		require.NotNil(t, step.Default)
		require.Empty(t, step.Default)
	}
}

// TestSwitch_EmptyCase verifies that a case with an empty list runs nothing, rather than the default
func TestSwitch_EmptyCase(t *testing.T) {

	step, err := NewSwitch(mapof.Any{
		"expression": "{{.Label}}",
		"cases":      map[string]any{"A": []any{}},
		"default":    map[string]any{"do": "halt"},
	}, nil)

	require.Nil(t, err)
	require.Contains(t, step.Cases, "A")
	require.Empty(t, step.Cases["A"])
}

// TestSwitch_RequiredRoles verifies that roles from the default and every case are merged
func TestSwitch_RequiredRoles(t *testing.T) {

	step, err := NewSwitch(mapof.Any{
		"expression": "{{.Label}}",
		"cases": map[string]any{
			"A": map[string]any{"do": "set-sharing", "role": "viewer", "group": "anonymous"},
		},
		"default": map[string]any{"do": "set-sharing", "role": "editor", "group": "anonymous"},
	}, nil)

	require.Nil(t, err)
	require.Equal(t, []string{"editor", "viewer"}, step.RequiredRoles())
}

// TestSwitch_Invalid verifies that every malformed configuration is refused when the Template loads
func TestSwitch_Invalid(t *testing.T) {

	halt := map[string]any{"do": "halt"}

	tests := []struct {
		name     string
		stepInfo mapof.Any
	}{
		{name: "missing expression", stepInfo: mapof.Any{"cases": map[string]any{"A": halt}}},
		{name: "empty expression", stepInfo: mapof.Any{"expression": "", "cases": map[string]any{"A": halt}}},
		{name: "unparseable expression", stepInfo: mapof.Any{"expression": "{{ .Unclosed", "cases": map[string]any{"A": halt}}},
		{name: "missing cases", stepInfo: mapof.Any{"expression": "{{.Label}}"}},
		{name: "empty cases", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": map[string]any{}}},
		{name: "cases is a list", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": []any{halt}}},
		{name: "cases is a string", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": "A"}},
		{name: "case is null", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": map[string]any{"A": nil}}},
		{name: "case is a string", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": map[string]any{"A": "halt"}}},
		{name: "case list holds a string", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": map[string]any{"A": []any{halt, "halt"}}}},
		{name: "case step is unknown", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": map[string]any{"A": map[string]any{"do": "nonexistent-step"}}}},
		{name: "case list step is unknown", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": map[string]any{"A": []any{map[string]any{"do": "nonexistent-step"}}}}},
		{name: "default is a string", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": map[string]any{"A": halt}, "default": "halt"}},
		{name: "default step is unknown", stepInfo: mapof.Any{"expression": "{{.Label}}", "cases": map[string]any{"A": halt}, "default": map[string]any{"do": "nonexistent-step"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewSwitch(test.stepInfo, nil)
			require.NotNil(t, err)
		})
	}
}

// TestSwitch_New verifies that "do":"switch" is registered with the step factory
func TestSwitch_New(t *testing.T) {

	result, err := New(mapof.Any{
		"do":         "switch",
		"expression": "{{.Label}}",
		"cases":      map[string]any{"A": map[string]any{"do": "halt"}},
	}, nil)

	require.Nil(t, err)
	require.IsType(t, Switch{}, result)
}
