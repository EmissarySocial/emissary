package build

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// StepStartupCreateStreams.Post reads nothing but the Domain's Theme, and a Domain builder
// resolves its Theme through the Theme service, whose registry is unexported and therefore
// cannot be seeded from this package.  What the Step then seeds is covered where it lives, in
// service.Stream.Startup; what remains testable here is the guard that stands in front of it.

// TestStepStartupCreateStreams_Post_RequiresDomainBuilder covers the runtime backstop behind
// RequiredModel/RequiredTemplateRoles.  Template validation rejects any Template that could reach
// this Step with the wrong builder, so a failure here means BOTH gates are gone -- and the Step
// would go on to seed content from a Theme that the current builder never chose.
func TestStepStartupCreateStreams_Post_RequiresDomainBuilder(t *testing.T) {

	// stubStartupTaskBuilder is any Builder that is not a build.Domain.
	step := StepStartupCreateStreams{}
	behavior := step.Post(stubStartupTaskBuilder{}, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "a non-Domain builder must halt the pipeline")
	require.Error(t, result.Error)
}

// TestStepStartupCreateStreams_Get pins the Step as a POST-only operation: seeding a Domain writes
// to the database, so a GET (a page build, a crawler, a prefetch) must do nothing at all.
func TestStepStartupCreateStreams_Get(t *testing.T) {

	step := StepStartupCreateStreams{}
	require.Nil(t, step.Get(stubStartupTaskBuilder{}, io.Discard))
}
