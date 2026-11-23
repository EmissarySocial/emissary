package build

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// StepStartupCreateGroups.Post reads nothing but the Domain's Theme, which a Domain builder resolves
// through the Theme service.  That registry is unexported, so what remains testable here is the
// guard in front of it; what the Step seeds is covered in service.Group.Startup.

// A Builder that is not a build.Domain halts the pipeline instead of seeding Groups.
func TestStepStartupCreateGroups_Post_RequiresDomainBuilder(t *testing.T) {

	step := StepStartupCreateGroups{}
	behavior := step.Post(stubStartupTaskBuilder{}, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "a non-Domain builder must halt the pipeline")
	require.Error(t, result.Error)
}

// GET is a no-op: seeding a Domain writes to the database, so it must not happen on a read.
func TestStepStartupCreateGroups_Get(t *testing.T) {

	step := StepStartupCreateGroups{}
	require.Nil(t, step.Get(stubStartupTaskBuilder{}, io.Discard))
}
