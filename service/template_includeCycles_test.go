package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// newIncludeTemplate returns a Template whose actions each include the actions named for them
func newIncludeTemplate(includes map[string][]string) model.Template {

	template := model.NewTemplate("test-template", nil)

	for actionID, includedIDs := range includes {
		action := model.NewAction()

		for _, includedID := range includedIDs {
			action.Steps = append(action.Steps, step.Include{Action: includedID})
		}

		template.Actions[actionID] = action
	}

	return template
}

// TestIncludeCycles verifies that an action which reaches itself through include is reported at load
func TestIncludeCycles(t *testing.T) {

	details := func(err derp.Error) []any {
		return err.Details
	}

	t.Run("no includes", func(t *testing.T) {
		template := newIncludeTemplate(map[string][]string{"view": {}, "edit": {}})
		require.Empty(t, includeCycles(&template))
	})

	t.Run("a shared action included twice is not a cycle", func(t *testing.T) {
		template := newIncludeTemplate(map[string][]string{"view": {"left", "right"}, "left": {"shared"}, "right": {"shared"}, "shared": {}})
		require.Empty(t, includeCycles(&template))
	})

	t.Run("an unknown action leads nowhere", func(t *testing.T) {
		template := newIncludeTemplate(map[string][]string{"view": {"missing"}})
		require.Empty(t, includeCycles(&template))
	})

	t.Run("an action that includes itself", func(t *testing.T) {
		template := newIncludeTemplate(map[string][]string{"view": {"view"}})
		errors := includeCycles(&template)
		require.Len(t, errors, 1)
		require.Contains(t, details(errors[0]), "path: view > view")
		require.Contains(t, details(errors[0]), "template: test-template")
	})

	t.Run("a cycle through other actions names every action on it", func(t *testing.T) {
		template := newIncludeTemplate(map[string][]string{"a": {"b"}, "b": {"c"}, "c": {"a"}, "outside": {"a"}})
		errors := includeCycles(&template)
		require.Len(t, errors, 3)
		require.Contains(t, details(errors[0]), "path: a > b > c > a")
		require.Contains(t, details(errors[1]), "path: b > c > a > b")
		require.Contains(t, details(errors[2]), "path: c > a > b > c")
	})

	t.Run("a cycle hidden in a branch", func(t *testing.T) {
		template := newIncludeTemplate(map[string][]string{"view": {}})
		action := template.Actions["view"]
		action.Steps = []step.Step{step.Switch{Cases: map[string][]step.Step{"x": {step.IfCondition{Then: []step.Step{step.Include{Action: "view"}}}}}}}
		template.Actions["view"] = action

		require.Len(t, includeCycles(&template), 1)
	})
}
