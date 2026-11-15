package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/stretchr/testify/require"
)

// TestPipeline_Includes finds an include of the action at any depth of a pipeline, and nothing else.
// This is the bound on what add-stream runs on an unsaved child before its Save.
func TestPipeline_Includes(t *testing.T) {

	photos := step.Include{Action: "photos"}

	check := func(name string, steps []step.Step, want bool) {
		t.Helper()
		require.Equal(t, want, Pipeline(steps).Includes("photos"), name)
	}

	check("a direct include", []step.Step{photos}, true)
	check("inside a modal", []step.Step{step.AsModal{SubSteps: []step.Step{step.Save{}, photos}}}, true)
	check("inside a modal's head", []step.Step{step.AsModal{Head: []step.Step{photos}}}, true)
	check("inside a condition's branch", []step.Step{step.IfCondition{Then: []step.Step{photos}}}, true)
	check("inside a condition's other branch", []step.Step{step.IfCondition{Otherwise: []step.Step{photos}}}, true)
	check("inside a switch case", []step.Step{step.Switch{Cases: map[string][]step.Step{"gallery": {photos}}}}, true)
	check("inside a switch default", []step.Step{step.Switch{Default: []step.Step{photos}}}, true)
	check("two levels down", []step.Step{step.AsModal{SubSteps: []step.Step{step.IfCondition{Then: []step.Step{photos}}}}}, true)
	check("another action", []step.Step{step.Include{Action: "covers"}}, false)
	check("the action's own step is not an include", []step.Step{step.EditAttachments{Fieldname: "photos"}}, false)
	check("no steps", nil, false)
}

// TestPipeline_EditsAttachments finds an edit-attachments step for the field at any depth of a
// pipeline, and nothing else. This is how the step tells the widget's own request from a Save through include.
func TestPipeline_EditsAttachments(t *testing.T) {

	own := step.EditAttachments{Category: "photo", Fieldname: "photos"}

	check := func(name string, steps []step.Step, want bool) {
		t.Helper()
		require.Equal(t, want, Pipeline(steps).EditsAttachments("photos"), name)
	}

	check("a direct step", []step.Step{own}, true)
	check("inside a modal", []step.Step{step.AsModal{SubSteps: []step.Step{own, step.Save{}}}}, true)
	check("inside a condition's branch", []step.Step{step.IfCondition{Then: []step.Step{own}}}, true)
	check("two levels down", []step.Step{step.AsModal{SubSteps: []step.Step{step.IfCondition{Otherwise: []step.Step{own}}}}}, true)
	check("another field", []step.Step{step.EditAttachments{Category: "cover", Fieldname: "covers"}}, false)
	check("an include of the action is not the step", []step.Step{step.Include{Action: "photos"}}, false)
	check("no steps", nil, false)
}

// TestNestedPipelines returns every pipeline a step holds, and nothing for a step that holds none
func TestNestedPipelines(t *testing.T) {

	leaf := step.Include{Action: "a"}

	require.Empty(t, nestedPipelines(leaf))
	require.Empty(t, nestedPipelines(nil))

	modal := step.AsModal{SubSteps: []step.Step{leaf}, Head: []step.Step{leaf, leaf}}
	require.ElementsMatch(t, []Pipeline{{leaf}, {leaf, leaf}}, nestedPipelines(modal))

	// An empty branch is not a pipeline
	condition := step.IfCondition{Then: []step.Step{leaf}}
	require.Equal(t, []Pipeline{{leaf}}, nestedPipelines(condition))

	cases := step.Switch{Cases: map[string][]step.Step{"x": {leaf}, "y": {}}, Default: []step.Step{leaf, leaf}}
	require.ElementsMatch(t, []Pipeline{{leaf}, {leaf, leaf}}, nestedPipelines(cases))
}
