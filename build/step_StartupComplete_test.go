package build

import (
	"io"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/stretchr/testify/require"
)

// StepStartupComplete writes to the Domain, so its guard must stop BEFORE the record is saved.  The
// stub builder carries a nil session, so a guard that fails to stop reaches Domain.Save() and panics
// on the nil collection rather than quietly passing.

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
}

// factory implements the Builder interface, returning this stub's factory
func (b stubStartupCompleteBuilder) factory() Factory { return b.factoryValue }

// session implements the Builder interface. The stub owns no database session.
func (b stubStartupCompleteBuilder) session() data.Session { return nil }

// newStartupCompleteBuilder wires a stub builder around a Domain service holding the provided record.
// Domain.Get() returns a pointer into the service's cache, so the fixture is installed through it,
// and that same pointer is returned so a test can assert on what the Step did (or did not) write.
func newStartupCompleteBuilder(domain model.Domain) (stubStartupCompleteBuilder, *model.Domain) {

	domainService := &service.Domain{}
	cached := domainService.Get()
	*cached = domain

	builder := stubStartupCompleteBuilder{
		factoryValue: stubStartupCompleteFactory{domainService: domainService},
	}

	return builder, cached
}

// runStartupCompleteStep executes the Step and collects its effect on the pipeline.
func runStartupCompleteStep(builder Builder) PipelineResult {

	behavior := StepStartupComplete{}.Post(builder, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	return result
}

// A Domain that has already finished setting up is left alone, and the Step does not fail the
// action that contains it.
func TestStepStartupComplete_Post_IgnoresLiveDomain(t *testing.T) {

	domain := model.NewDomain()
	domain.StateID = model.DomainStateLive

	builder, cached := newStartupCompleteBuilder(domain)
	result := runStartupCompleteStep(builder)

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Equal(t, model.DomainStateLive, cached.StateID)
}

// GET is a no-op: ending setup writes to the database, so it must not happen on a read.
func TestStepStartupComplete_Get_DoesNothing(t *testing.T) {

	domain := model.NewDomain() // NewDomain starts in the STARTUP state
	require.Equal(t, model.DomainStateStartup, domain.StateID)

	builder, cached := newStartupCompleteBuilder(domain)
	require.Nil(t, StepStartupComplete{}.Get(builder, io.Discard))
	require.Equal(t, model.DomainStateStartup, cached.StateID, "GET must not move the Domain out of STARTUP")
}
