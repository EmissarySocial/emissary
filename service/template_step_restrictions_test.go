package service

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	modelStep "github.com/EmissarySocial/emissary/model/step"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
)

// newStepRestrictionTemplate builds the smallest Template that reaches the per-step validation
// rules in validateTemplates: a category, one state, one action, and one step inside it.
func newStepRestrictionTemplate(templateID string, templateRole string, modelName string, step modelStep.Step) model.Template {

	action := model.NewAction()
	action.Steps = sliceof.NewObject[modelStep.Step]()
	action.Steps = append(action.Steps, step)

	result := model.NewTemplate(templateID, nil)
	result.Category = "Test"
	result.TemplateRole = templateRole
	result.Model = modelName
	result.States["default"] = model.NewState()
	result.Actions["seed"] = action

	// Inherit the model's base schema, exactly as service.Template.Add does.  Without it,
	// validateTemplates panics on a nil schema before it reaches the step rules.
	result.Schema = schema.New(result.BaseSchema())

	return result
}

// validateOneTemplate runs the real validateTemplates() pass over a single hand-built Template.
// validateTemplates reads nothing but templatePrep, so no factory or filesystem is required.
func validateOneTemplate(template model.Template) sliceof.Object[derp.Error] {

	service := &Template{
		templatePrep: set.Map[model.Template]{template.TemplateID: template},
	}

	return service.validateTemplates()
}

// containsValidationError reports whether any of the errors mentions the given substring.
func containsValidationError(errors sliceof.Object[derp.Error], substring string) bool {

	for _, err := range errors {
		if strings.Contains(err.Error(), substring) {
			return true
		}
	}

	return false
}

// TestStartupCreateStreams_RequiresAdminTemplate asserts the load-time guard that keeps
// "startup-create-streams" inside the admin console.  The step seeds a Domain with content, so a
// Template that is not an admin Template must refuse to load rather than fail at request time.
func TestStartupCreateStreams_RequiresAdminTemplate(t *testing.T) {

	const roleMessage = "Step can only be used in Templates with a specific templateRole"
	const modelMessage = "Step can only be used in specific Templates"

	step, err := modelStep.New(map[string]any{"do": "startup-create-streams"})
	require.Nil(t, err)

	// An admin Template that builds the Domain model is the one permitted combination.
	{
		errors := validateOneTemplate(newStepRestrictionTemplate("admin-ok", "admin", "Domain", step))
		require.False(t, containsValidationError(errors, roleMessage), "admin/Domain template must satisfy the templateRole rule")
		require.False(t, containsValidationError(errors, modelMessage), "admin/Domain template must satisfy the model rule")
	}

	// Right model, wrong role.  This is the case that RequiredModel alone would let through:
	// several Templates build a Domain, and only the admin ones may seed one.
	{
		errors := validateOneTemplate(newStepRestrictionTemplate("public-domain", "search", "Domain", step))
		require.True(t, containsValidationError(errors, roleMessage), "non-admin template must be rejected")
	}

	// Right role, wrong model.
	{
		errors := validateOneTemplate(newStepRestrictionTemplate("admin-stream", "admin", "Stream", step))
		require.True(t, containsValidationError(errors, modelMessage), "non-Domain template must be rejected")
	}
}

// TestStartupComplete_RequiresAdminTemplate asserts the same load-time guard around
// "startup-complete".  This step ends the startup wizard and opens the Domain to the public, so a
// Template that is not an admin Template must refuse to load rather than fail at request time.
func TestStartupComplete_RequiresAdminTemplate(t *testing.T) {

	const roleMessage = "Step can only be used in Templates with a specific templateRole"
	const modelMessage = "Step can only be used in specific Templates"

	step, err := modelStep.New(map[string]any{"do": "startup-complete"})
	require.Nil(t, err)

	// An admin Template that builds the Domain model is the one permitted combination.
	{
		errors := validateOneTemplate(newStepRestrictionTemplate("admin-ok", "admin", "Domain", step))
		require.False(t, containsValidationError(errors, roleMessage), "admin/Domain template must satisfy the templateRole rule")
		require.False(t, containsValidationError(errors, modelMessage), "admin/Domain template must satisfy the model rule")
	}

	// Right model, wrong role.
	{
		errors := validateOneTemplate(newStepRestrictionTemplate("public-domain", "search", "Domain", step))
		require.True(t, containsValidationError(errors, roleMessage), "non-admin template must be rejected")
	}

	// Right role, wrong model.
	{
		errors := validateOneTemplate(newStepRestrictionTemplate("admin-stream", "admin", "Stream", step))
		require.True(t, containsValidationError(errors, modelMessage), "non-Domain template must be rejected")
	}
}
