package build

import (
	"io"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/stretchr/testify/require"
)

// StepStartupComplete writes to the Domain, so its guard must stop BEFORE the record is read or
// saved.  The stub builder's session is whatever a test hands it: nil for a guard test, so a guard
// that fails to stop reaches Domain.Load() and panics rather than quietly passing.

// stubStartupCompleteFactory is a build.Factory offering the one service the Step reads: the Domain
// singleton.  Every other method is inherited from the embedded (nil) interface.
type stubStartupCompleteFactory struct {
	Factory
	domainService *service.Domain
}

// Domain implements the build.Factory interface, returning this stub's Domain service
func (f stubStartupCompleteFactory) Domain() *service.Domain { return f.domainService }

// stubStartupCompleteBuilder is a build.Builder exposing only what StepStartupComplete.Post reaches.
type stubStartupCompleteBuilder struct {
	Builder
	factoryValue Factory
	sessionValue data.Session
}

// factory implements the Builder interface, returning this stub's factory
func (b stubStartupCompleteBuilder) factory() Factory { return b.factoryValue }

// session implements the Builder interface, returning this stub's session
func (b stubStartupCompleteBuilder) session() data.Session { return b.sessionValue }

// newStartupCompleteBuilder wires a stub builder around a Domain service whose record is stored in
// an in-memory database and published, so the Step reads the cache and loads the stored record.
func newStartupCompleteBuilder(t *testing.T, domain model.WritableDomain) (stubStartupCompleteBuilder, *service.Domain, data.Session) {

	t.Helper()

	domainService, session := newSeededDomainService(t, domain)

	builder := stubStartupCompleteBuilder{
		factoryValue: stubStartupCompleteFactory{domainService: domainService},
		sessionValue: session,
	}

	return builder, domainService, session
}

// runStartupCompleteStep executes the Step and collects its effect on the pipeline.
func runStartupCompleteStep(builder Builder) PipelineResult {

	behavior := StepStartupComplete{}.Post(builder, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	return result
}

// A Domain that has already finished setting up is left alone, and the Step does not fail the
// action that contains it.  The nil session proves the record is never read.
func TestStepStartupComplete_Post_IgnoresLiveDomain(t *testing.T) {

	domain := model.NewWritableDomain()
	domain.StateID = model.DomainStateLive

	builder, domainService, _ := newStartupCompleteBuilder(t, domain)
	builder.sessionValue = nil
	result := runStartupCompleteStep(builder)

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Equal(t, model.DomainStateLive, domainService.Get().StateID)
}

// A Domain still in STARTUP is moved to LIVE, in the database and in the cache.
func TestStepStartupComplete_Post_MovesTheDomainLive(t *testing.T) {

	builder, domainService, session := newStartupCompleteBuilder(t, model.NewWritableDomain())
	result := runStartupCompleteStep(builder)

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Equal(t, model.DomainStateLive, loadSeededDomain(t, session).StateID)
	require.Equal(t, model.DomainStateLive, domainService.Get().StateID)
}

// The stored record is checked again after it is loaded: when another request finished the wizard
// since the cache was read, nothing is written.
func TestStepStartupComplete_Post_IgnoresAStoredLiveDomain(t *testing.T) {

	builder, _, session := newStartupCompleteBuilder(t, model.NewWritableDomain())

	live := model.NewWritableDomain()
	live.StateID = model.DomainStateLive
	storeSeededDomain(t, session, live)
	before := loadSeededDomain(t, session)

	result := runStartupCompleteStep(builder)

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Equal(t, before.Revision, loadSeededDomain(t, session).Revision)
}

// A record that cannot be loaded halts the action rather than moving a blank Domain live.
func TestStepStartupComplete_Post_LoadFailureHalts(t *testing.T) {

	builder, domainService, _ := newStartupCompleteBuilder(t, model.NewWritableDomain())
	builder.sessionValue = failingDomainSession{}
	result := runStartupCompleteStep(builder)

	require.True(t, result.Halt)
	require.Error(t, result.Error)
	require.Equal(t, model.DomainStateStartup, domainService.Get().StateID)
}

// A record that cannot be saved halts the action and leaves the cache as it was.
func TestStepStartupComplete_Post_SaveFailureHalts(t *testing.T) {

	builder, domainService, session := newStartupCompleteBuilder(t, model.NewWritableDomain())

	invalid := model.NewWritableDomain()
	invalid.ColorMode = "NOT-A-COLOR-MODE"
	storeSeededDomain(t, session, invalid)

	result := runStartupCompleteStep(builder)

	require.True(t, result.Halt)
	require.Error(t, result.Error)
	require.Equal(t, model.DomainStateStartup, domainService.Get().StateID)
}

// GET is a no-op: ending setup writes to the database, so it must not happen on a read.
func TestStepStartupComplete_Get_DoesNothing(t *testing.T) {

	domain := model.NewWritableDomain() // NewDomain starts in the STARTUP state
	require.Equal(t, model.DomainStateStartup, domain.StateID)

	builder, domainService, _ := newStartupCompleteBuilder(t, domain)
	require.Nil(t, StepStartupComplete{}.Get(builder, io.Discard))
	require.Equal(t, model.DomainStateStartup, domainService.Get().StateID, "GET must not move the Domain out of STARTUP")
}
