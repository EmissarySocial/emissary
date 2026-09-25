package build

import (
	"io"
	"testing"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// StepWithStreamSource.execute loads a record and builds a sub-builder, which needs a live Factory,
// session, and Template -- none of which this package can assemble.  What IS reachable here is the
// builder-type guard in front of all of that, on both methods.

// TestStepWithStreamSource_RequiresStreamBuilder covers the guard that stands in front of the load.
// A StreamSource is found by the Stream it populates, so a non-Stream builder has nothing to look
// up -- and without this guard the Step would go on to dereference a Stream it never held.
func TestStepWithStreamSource_RequiresStreamBuilder(t *testing.T) {

	step := StepWithStreamSource{}

	for name, behavior := range map[string]PipelineBehavior{
		"GET":  step.Get(stubStartupTaskBuilder{}, io.Discard),
		"POST": step.Post(stubStartupTaskBuilder{}, io.Discard),
	} {
		t.Run(name, func(t *testing.T) {

			result := NewPipelineResult()
			behavior(&result)

			require.True(t, result.Halt, "a non-Stream builder must halt the pipeline")
			require.Error(t, result.Error)
		})
	}
}

// TestStepWithStreamSource_GuardsBothMethods pins that this Step does the same work on GET and
// POST.  A container has to run on GET so the settings form can RENDER, which is the opposite of
// the POST-only rule that leaf steps like with-children follow -- an easy thing to "fix" wrongly.
func TestStepWithStreamSource_GuardsBothMethods(t *testing.T) {

	step := StepWithStreamSource{}

	require.NotNil(t, step.Get(stubStartupTaskBuilder{}, io.Discard), "GET must reach the guard, not no-op")
	require.NotNil(t, step.Post(stubStartupTaskBuilder{}, io.Discard))
}

// TestStepWithStreamSource_CarriesItsSubSteps confirms that the parsed sub-pipeline survives the
// hand-off from model/step into build, which is the one thing the registration in step_.go does
func TestStepWithStreamSource_CarriesItsSubSteps(t *testing.T) {

	parsed, err := step.NewWithStreamSource(mapof.Any{
		"steps": []mapof.Any{{"do": "edit"}, {"do": "save"}},
	})

	require.Nil(t, err)

	built := StepWithStreamSource(parsed)
	require.Len(t, built.SubSteps, 2)
}
