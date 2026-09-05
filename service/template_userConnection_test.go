package service

import (
	"os"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/form"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

/******************************************
 * User Connection Forms
 *
 * The Mailchimp connection form exists twice, and the only thing
 * separating them is one `readOnly` on the Audience ID. That single
 * property is what makes the audience immutable (D43), and nothing
 * else in the codebase would notice if it disappeared.
 ******************************************/

// userSettingsTemplate parses the shipped user-settings definition
func userSettingsTemplate(t *testing.T) model.Template {

	t.Helper()

	definition, err := os.ReadFile("../_embed/templates/user-settings/template.hjson")
	require.NoError(t, err)

	result := model.NewTemplate("user-settings", nil)
	require.NoError(t, hjson.Unmarshal(definition, &result))

	return result
}

// audienceElement returns the Audience ID field from the named action's edit form
func audienceElement(t *testing.T, template model.Template, actionID string) form.Element {

	t.Helper()

	action, exists := template.Actions[actionID]
	require.True(t, exists, "action %q must exist", actionID)

	for _, step := range allSteps(action.Steps) {

		editStep, ok := step.(modelStep.EditModelObject)

		if !ok {
			continue
		}

		for _, element := range flattenElements(editStep.Form) {
			if element.Path == "data.audienceId" {
				return element
			}
		}
	}

	t.Fatalf("action %q has no data.audienceId field", actionID)
	return form.Element{}
}

// flattenElements walks a form tree.
//
// form.Element.AllElements is deliberately NOT used: it returns nothing for a read-only
// element, which is the upstream half of how `readOnly` is enforced -- and would make the
// very field these tests exist to check invisible to them.
func flattenElements(element form.Element) []form.Element {

	result := []form.Element{element}

	for _, child := range element.Children {
		result = append(result, flattenElements(child)...)
	}

	return result
}

// allSteps flattens a step tree so that a step nested inside as-modal and
// with-user-connection is still reachable
func allSteps(steps []modelStep.Step) []modelStep.Step {

	result := make([]modelStep.Step, 0, len(steps))

	for _, step := range steps {

		result = append(result, step)

		switch typed := step.(type) {

		case modelStep.AsModal:
			result = append(result, allSteps(typed.SubSteps)...)

		case modelStep.WithUserConnection:
			result = append(result, allSteps(typed.SubSteps)...)
		}
	}

	return result
}

// TestUserConnection_AudienceIsEditableOnlyWhenCreating pins the one property that makes
// the audience immutable
func TestUserConnection_AudienceIsEditableOnlyWhenCreating(t *testing.T) {

	// `form.SetURLValues` skips read-only elements before it reads anything from the request,
	// so this is real enforcement rather than presentation -- and it is the whole of it.
	// Without it, editing the audience strands the merge field and the webhook inside the
	// old one, with nothing able to address them again, and nothing reports it (D43).

	template := userSettingsTemplate(t)

	require.False(t, audienceElement(t, template, "connections-new-mailchimp").ReadOnly,
		"the audience must be settable when the connection is created")

	require.True(t, audienceElement(t, template, "connections-edit-mailchimp").ReadOnly,
		"the audience must be immutable once setup has run against it (D43)")
}

// TestUserConnection_BothFormsCarryTheSameCredentialFields guards the drift that a split
// form invites
func TestUserConnection_BothFormsCarryTheSameCredentialFields(t *testing.T) {

	// Two near-identical forms is the cost D43 accepted. A field added to one and not the
	// other is silent: the connection simply cannot be edited in that respect afterward.

	template := userSettingsTemplate(t)

	paths := func(actionID string) []string {

		action, exists := template.Actions[actionID]
		require.True(t, exists, "action %q must exist", actionID)

		result := make([]string, 0)

		for _, step := range allSteps(action.Steps) {
			if editStep, ok := step.(modelStep.EditModelObject); ok {
				for _, element := range flattenElements(editStep.Form) {
					if element.Path != "" {
						result = append(result, element.Path)
					}
				}
			}
		}

		return result
	}

	require.ElementsMatch(t, paths("connections-new-mailchimp"), paths("connections-edit-mailchimp"))
}
