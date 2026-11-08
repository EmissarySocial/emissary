package build

import (
	"io"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// stubIncludeBuilder is a build.Builder exposing only what StepInclude reaches: one action, "halts",
// whose only step halts the pipeline, so a halted result shows that the include ran
type stubIncludeBuilder struct {
	Builder
}

// actions implements the Builder interface, returning the one action this stub offers
func (b stubIncludeBuilder) actions() map[string]model.Action {
	return map[string]model.Action{
		"halts": {Steps: []step.Step{step.Halt{}}},
	}
}

// factory implements the Builder interface. The pipeline only passes the factory along, so none is needed.
func (b stubIncludeBuilder) factory() Factory { return unusedFactory{} }

// runInclude runs the Step for one request method and returns the result it produced
func runInclude(t *testing.T, include StepInclude, actionMethod ActionMethod) PipelineResult {

	t.Helper()

	var behavior PipelineBehavior

	if actionMethod == ActionMethodGet {
		behavior = include.Get(stubIncludeBuilder{}, io.Discard)
	} else {
		behavior = include.Post(stubIncludeBuilder{}, io.Discard)
	}

	result := NewPipelineResult()

	if behavior != nil {
		behavior(&result)
	}

	return result
}

// TestStepInclude_Method runs the included action for each method the include allows, and skips it for the other
func TestStepInclude_Method(t *testing.T) {

	ran := func(method string, actionMethod ActionMethod) bool {
		t.Helper()
		return runInclude(t, StepInclude{Action: "halts", Method: method}, actionMethod).Halt
	}

	require.True(t, ran("both", ActionMethodGet))
	require.True(t, ran("both", ActionMethodPost))

	require.True(t, ran("get", ActionMethodGet))
	require.False(t, ran("get", ActionMethodPost), "a GET-only include must not run on a POST")

	require.False(t, ran("post", ActionMethodGet), "a POST-only include must not run on a GET")
	require.True(t, ran("post", ActionMethodPost))

	// An include built in Go without a method runs on both, as one parsed with the default does
	require.True(t, ran("", ActionMethodGet))
	require.True(t, ran("", ActionMethodPost))
}

// TestStepInclude_MissingAction halts with a 400 on either method when the action does not exist
func TestStepInclude_MissingAction(t *testing.T) {

	for _, actionMethod := range []ActionMethod{ActionMethodGet, ActionMethodPost} {

		result := runInclude(t, StepInclude{Action: "missing", Method: "both"}, actionMethod)

		require.True(t, result.Halt)
		require.Error(t, result.Error)
		require.Equal(t, 400, derp.ErrorCode(result.Error))
	}
}

// TestStepInclude_SkippedActionIsNeverLookedUp confirms that a skipped include never reads the
// builder, so a POST-only include that names no real action does nothing on a GET
func TestStepInclude_SkippedActionIsNeverLookedUp(t *testing.T) {

	result := runInclude(t, StepInclude{Action: "missing", Method: "post"}, ActionMethodGet)
	require.False(t, result.Halt)
	require.NoError(t, result.Error)
}
