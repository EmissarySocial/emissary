package build

import (
	"io"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// StepWithChildren.Post walks a Stream's siblings and runs a sub-pipeline against each one, which
// needs a live Factory, session, and Stream records -- none of which this package can assemble.
// What IS reachable here is the builder-type guard in front of that loop, plus the PipelineResult
// contract the loop depends on to notice a child that failed.  Both are covered below.

// TestStepWithChildren_Post_RequiresStreamBuilder covers the guard that stands in front of the
// loop.  Template validation should keep a non-Stream builder from ever reaching this Step, so a
// failure here means the Step would go on to dereference a Stream that the builder never held.
func TestStepWithChildren_Post_RequiresStreamBuilder(t *testing.T) {

	// stubStartupTaskBuilder is any Builder that is not a build.Stream.
	step := StepWithChildren{}
	behavior := step.Post(stubStartupTaskBuilder{}, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "a non-Stream builder must halt the pipeline")
	require.Error(t, result.Error)
}

// TestStepWithChildren_Get pins the Step as a POST-only operation: running sub-steps against every
// child writes to the database, so a GET (a page build, a crawler, a prefetch) must do nothing.
func TestStepWithChildren_Get(t *testing.T) {
	step := StepWithChildren{}
	require.Nil(t, step.Get(stubStartupTaskBuilder{}, io.Discard), "GET must be a no-op")
}

// TestPipelineResult_Merge_CarriesHaltAndError pins the invariant that StepWithChildren.Post relies
// on to stop after a child fails.  The Step merges each child's result into its accumulator and
// then tests the accumulator, so Merge MUST carry both signals across.  If it stopped doing so, a
// failing child would be silently skipped and the loop would run on through its siblings.
func TestPipelineResult_Merge_CarriesHaltAndError(t *testing.T) {

	accumulator := NewPipelineResult()
	require.False(t, accumulator.Halt)
	require.Nil(t, accumulator.Error)

	child := NewPipelineResult()
	child.Halt = true
	child.Error = derp.Internal("test", "child pipeline failed")

	accumulator.Merge(child)

	require.True(t, accumulator.Halt, "a child's Halt must reach the accumulator")
	require.Error(t, accumulator.Error, "a child's Error must reach the accumulator")
}

// TestPipelineResult_Merge_KeepsHalt confirms that a later, successful child cannot clear a Halt
// that an earlier child already set -- the accumulator only ever latches Halt on.
func TestPipelineResult_Merge_KeepsHalt(t *testing.T) {

	accumulator := NewPipelineResult()
	accumulator.Halt = true

	accumulator.Merge(NewPipelineResult())

	require.True(t, accumulator.Halt, "Halt must not be cleared by a subsequent success")
}
