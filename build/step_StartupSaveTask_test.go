package build

import (
	"io"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/form"
	"github.com/stretchr/testify/require"
)

// StepStartupSaveTask writes to the Domain, so each of its guards must stop BEFORE the record is
// saved.  These tests assert that the Domain is left untouched -- and because the stub builder
// carries a nil session, a guard that fails to stop reaches Domain.Save() and panics on the nil
// collection rather than quietly passing.

// stubStartupTaskFactory is a build.Factory offering the one service the Step reads: the Domain
// singleton.  Every other method is inherited from the embedded (nil) interface.
type stubStartupTaskFactory struct {
	Factory
	domainService *service.Domain
}

// Domain implements the build.Factory interface, returning this stub's Domain service
func (f stubStartupTaskFactory) Domain() *service.Domain { return f.domainService }

// stubStartupTaskBuilder is a build.Builder exposing only what StepStartupSaveTask.Post reaches.
// Theme() returns a hand-built Theme rather than delegating to the Theme service, whose registry
// is unexported and therefore cannot be seeded from this package.
type stubStartupTaskBuilder struct {
	Builder
	factoryValue Factory
	theme        model.Theme
}

// factory implements the Builder interface, returning this stub's factory
func (b stubStartupTaskBuilder) factory() Factory { return b.factoryValue }

// session implements the Builder interface. The stub owns no database session.
func (b stubStartupTaskBuilder) session() data.Session { return nil }

// ThemeID implements the Builder interface, returning this stub's theme ID
func (b stubStartupTaskBuilder) ThemeID() string { return b.theme.ThemeID }

// Theme implements the Builder interface, returning this stub's Theme
func (b stubStartupTaskBuilder) Theme(_ string) model.Theme { return b.theme }

// newStartupTaskTheme builds a Theme whose startup tasks carry the provided values.
func newStartupTaskTheme(values ...string) model.Theme {

	theme := model.NewTheme("test", nil)

	for _, value := range values {
		theme.StartupTasks = append(theme.StartupTasks, form.LookupCode{Value: value})
	}

	return theme
}

// newStartupTaskBuilder wires a stub builder around a Domain service holding the provided record.
// Domain.Get() returns a pointer into the service's cache, so the fixture is installed through it,
// and that same pointer is returned so a test can assert on what the Step did (or did not) write.
func newStartupTaskBuilder(domain model.Domain, theme model.Theme) (stubStartupTaskBuilder, *model.Domain) {

	domainService := &service.Domain{}
	cached := domainService.Get()
	*cached = domain

	builder := stubStartupTaskBuilder{
		factoryValue: stubStartupTaskFactory{domainService: domainService},
		theme:        theme,
	}

	return builder, cached
}

// runStartupTaskStep executes the Step and collects its effect on the pipeline.
func runStartupTaskStep(builder Builder, value string) PipelineResult {

	step := StepStartupSaveTask{Value: value}
	behavior := step.Post(builder, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	return result
}

// Once a Domain is live, the startup wizard is over and tasks are no longer recorded -- even for
// a task the Theme does define.
func TestStepStartupSaveTask_Post_IgnoresLiveDomain(t *testing.T) {

	domain := model.NewDomain()
	domain.StateID = model.DomainStateLive

	builder, cached := newStartupTaskBuilder(domain, newStartupTaskTheme("sample-content"))
	result := runStartupTaskStep(builder, "sample-content")

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Empty(t, cached.StartupTasks, "a live Domain must not collect startup tasks")
}

// A task the Theme does not define is dropped, not recorded -- otherwise a renamed or mistyped
// task would accumulate in the Domain as a value that nothing can display.
func TestStepStartupSaveTask_Post_IgnoresUnknownTask(t *testing.T) {

	domain := model.NewDomain() // NewDomain starts in the STARTUP state
	require.Equal(t, model.DomainStateStartup, domain.StateID)

	builder, cached := newStartupTaskBuilder(domain, newStartupTaskTheme("some-other-task"))
	result := runStartupTaskStep(builder, "sample-content")

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Empty(t, cached.StartupTasks, "a task the Theme does not define must not be recorded")
}

// A task that is already recorded is not written again.  The Theme here DOES define the task, so
// this reaches the duplicate check rather than stopping at the one before it.
func TestStepStartupSaveTask_Post_IgnoresDuplicateTask(t *testing.T) {

	domain := model.NewDomain()
	domain.StartupTasks = append(domain.StartupTasks, "sample-content")

	builder, cached := newStartupTaskBuilder(domain, newStartupTaskTheme("sample-content"))
	result := runStartupTaskStep(builder, "sample-content")

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Equal(t, []string{"sample-content"}, []string(cached.StartupTasks), "an already-recorded task must not be added twice")
}
