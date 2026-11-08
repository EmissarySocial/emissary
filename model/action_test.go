package model

import (
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestUnmarshalActions_DefaultsViewFiles decodes a Template the way the loader does, and confirms that
// every view that names no file renders its own action's file, at any depth
func TestUnmarshalActions_DefaultsViewFiles(t *testing.T) {

	definition := `{
		"templateId": "sample",
		"actions": {
			"view":      {"steps": [{"do": "view-html"}]},
			"stylesheet":{"steps": [{"do": "view-css"}]},
			"modal":     {"steps": [{"do": "as-modal", "head": [{"do": "view-html"}], "steps": [{"do": "view-html", "method": "both"}]}]},
			"branch":    {"steps": [{"do": "if", "condition": "{{true}}", "then": [{"do": "view-html"}], "else": [{"do": "view-css"}]}]},
			"choose":    {"steps": [{"do": "switch", "expression": "x", "cases": {"one": {"do": "view-html"}, "many": [{"do": "view-html"}]}, "default": {"do": "view-html"}}]},
			"parent":    {"steps": [{"do": "with-parent", "steps": [{"do": "view-html"}]}]},
			"named":     {"steps": [{"do": "view-html", "file": "elsewhere"}, {"do": "view-css", "file": "styles"}]},
			"blank":     {"steps": [{"do": "view-html", "file": ""}]},
			"caller":    {"steps": [{"do": "include", "action": "fragment"}]},
			"fragment":  {"steps": [{"do": "view-html", "method": "get"}]}
		}
	}`

	template := NewTemplate("sample", nil)
	require.NoError(t, json.Unmarshal([]byte(definition), &template))

	steps := func(actionID string) []step.Step {
		t.Helper()
		action, exists := template.Action(actionID)
		require.True(t, exists, "action %q", actionID)
		return action.Steps
	}

	// A view at the top of an action, of either kind
	require.Equal(t, "view", steps("view")[0].(step.ViewHTML).File)
	require.Equal(t, "stylesheet", steps("stylesheet")[0].(step.ViewCSS).File)

	// Inside a modal's head and body
	modal := steps("modal")[0].(step.AsModal)
	require.Equal(t, "modal", modal.Head[0].(step.ViewHTML).File)
	require.Equal(t, "modal", modal.SubSteps[0].(step.ViewHTML).File)
	require.Equal(t, "both", modal.SubSteps[0].(step.ViewHTML).Method, "the default must not disturb the step's other settings")

	// Inside both branches of a condition
	branch := steps("branch")[0].(step.IfCondition)
	require.Equal(t, "branch", branch.Then[0].(step.ViewHTML).File)
	require.Equal(t, "branch", branch.Otherwise[0].(step.ViewCSS).File)

	// Inside a switch case written as one step, a case written as a list, and the default
	choose := steps("choose")[0].(step.Switch)
	require.Equal(t, "choose", onlyViewFile(t, choose.Cases["one"]))
	require.Equal(t, "choose", onlyViewFile(t, choose.Cases["many"]))
	require.Equal(t, "choose", onlyViewFile(t, choose.Default))

	// Inside a with-* step, whose name is the one the running builder would have used
	require.Equal(t, "parent", steps("parent")[0].(step.WithParent).SubSteps[0].(step.ViewHTML).File)

	// A named file is never replaced, and an empty one is treated as no name at all
	require.Equal(t, "elsewhere", steps("named")[0].(step.ViewHTML).File)
	require.Equal(t, "styles", steps("named")[1].(step.ViewCSS).File)
	require.Equal(t, "blank", steps("blank")[0].(step.ViewHTML).File)

	// An included action keeps its own file, whoever includes it
	require.Equal(t, "fragment", steps("fragment")[0].(step.ViewHTML).File)
	require.Equal(t, "fragment", steps("caller")[0].(step.Include).Action)
}

// onlyViewFile returns the file of the view-html that a branch holds as its only step
func onlyViewFile(t *testing.T, steps []step.Step) string {

	t.Helper()

	if len(steps) != 1 {
		t.Fatalf("branch holds %d steps, want 1", len(steps))
		return ""
	}

	view, isView := steps[0].(step.ViewHTML)
	require.True(t, isView, "the branch's step is %q, not view-html", steps[0].Name())

	return view.File
}

// TestUnmarshalActions_DefaultsRegistrationViewFiles confirms that a Registration's actions get the same default
func TestUnmarshalActions_DefaultsRegistrationViewFiles(t *testing.T) {

	registration := NewRegistration("sample", nil)
	require.NoError(t, json.Unmarshal([]byte(`{"actions": {"signup": {"steps": [{"do": "view-html"}]}}}`), &registration))

	action, exists := registration.Action("signup")
	require.True(t, exists)
	require.Equal(t, "signup", action.Steps[0].(step.ViewHTML).File)
}

// TestDefaultViewFiles covers every shape of raw step data the walk accepts, and the values it must leave alone
func TestDefaultViewFiles(t *testing.T) {

	view := func() map[string]any { return map[string]any{"do": "view-html"} }

	t.Run("map", func(t *testing.T) {
		value := view()
		defaultViewFiles(value, "action")
		require.Equal(t, "action", value["file"])
	})

	t.Run("mapof.Any", func(t *testing.T) {
		value := mapof.Any(view())
		defaultViewFiles(value, "action")
		require.Equal(t, "action", value["file"])
	})

	t.Run("slice of any", func(t *testing.T) {
		value := view()
		defaultViewFiles([]any{"not a step", 42, nil, value}, "action")
		require.Equal(t, "action", value["file"])
	})

	t.Run("slice of maps", func(t *testing.T) {
		value := view()
		defaultViewFiles([]map[string]any{value}, "action")
		require.Equal(t, "action", value["file"])
	})

	t.Run("slice of mapof.Any", func(t *testing.T) {
		value := mapof.Any(view())
		defaultViewFiles([]mapof.Any{value}, "action")
		require.Equal(t, "action", value["file"])
	})

	t.Run("nested under any key", func(t *testing.T) {
		value := view()
		defaultViewFiles(map[string]any{"do": "with-parent", "steps": []any{map[string]any{"do": "if", "then": []any{value}}}}, "action")
		require.Equal(t, "action", value["file"])
	})

	t.Run("values that hold no steps", func(t *testing.T) {
		require.NotPanics(t, func() {
			defaultViewFiles(nil, "action")
			defaultViewFiles("view-html", "action")
			defaultViewFiles(3.14, "action")
			defaultViewFiles([]any{}, "action")
			defaultViewFiles(map[string]any{}, "action")
			defaultViewFiles([]map[string]any(nil), "action")
			defaultViewFiles(map[string]any(nil), "action")
			defaultViewFiles([]any{map[string]any(nil)}, "action")
		})
	})

	t.Run("other steps are untouched", func(t *testing.T) {
		value := map[string]any{"do": "view-json", "value": "{{.}}"}
		defaultViewFiles(value, "action")
		require.NotContains(t, value, "file")
	})
}

// TestNeedsViewFile names which raw steps take the default
func TestNeedsViewFile(t *testing.T) {

	check := func(stepInfo map[string]any, want bool) {
		t.Helper()
		require.Equal(t, want, needsViewFile(stepInfo), "%v", stepInfo)
	}

	check(map[string]any{"do": "view-html"}, true)
	check(map[string]any{"do": "view-css"}, true)
	check(map[string]any{"do": "view-html", "file": ""}, true)
	check(map[string]any{"do": "view-html", "file": nil}, true)
	check(map[string]any{"do": "view-html", "file": "detail"}, false)
	check(map[string]any{"do": "view-css", "file": "styles"}, false)
	check(map[string]any{"do": "view-json"}, false)
	check(map[string]any{"do": "VIEW-HTML"}, false)
	check(map[string]any{"do": 7}, false)
	check(map[string]any{"file": "detail"}, false)
	check(map[string]any{}, false)
	check(nil, false)
}

// FuzzDefaultViewFiles confirms that the walk never panics on any decoded JSON, leaves every view
// step naming a file, and changes nothing when it runs a second time
func FuzzDefaultViewFiles(f *testing.F) {

	f.Add(`[{"do": "view-html"}]`)
	f.Add(`[{"do": "as-modal", "steps": [{"do": "view-css", "file": ""}]}]`)
	f.Add(`{"do": "switch", "cases": {"a": {"do": "view-html"}}, "default": [{"do": "view-html", "file": "x"}]}`)
	f.Add(`[1, "two", null, true, {"do": 3}]`)
	f.Add(`{}`)

	f.Fuzz(func(t *testing.T, input string) {

		var value any

		if err := json.Unmarshal([]byte(input), &value); err != nil {
			return
		}

		defaultViewFiles(value, "action")

		once, err := json.Marshal(value)
		require.NoError(t, err)

		defaultViewFiles(value, "action")

		twice, err := json.Marshal(value)
		require.NoError(t, err)
		require.Equal(t, string(once), string(twice), "a second walk changed the data")

		requireNoUnnamedViews(t, value)
	})
}

// requireNoUnnamedViews fails if any view step in raw step data still names no file
func requireNoUnnamedViews(t *testing.T, value any) {

	t.Helper()

	switch typed := value.(type) {

	case map[string]any:
		require.False(t, needsViewFile(typed), "unnamed view left behind: %v", typed)

		for _, item := range typed {
			requireNoUnnamedViews(t, item)
		}

	case []any:
		for _, item := range typed {
			requireNoUnnamedViews(t, item)
		}
	}
}
