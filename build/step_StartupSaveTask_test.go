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
// read or saved.  The stub builder's session is whatever a test hands it: nil for a guard test, so
// a guard that fails to stop reaches Domain.Load() and panics rather than quietly passing.

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
	sessionValue data.Session
	theme        model.Theme
}

// factory implements the Builder interface, returning this stub's factory
func (b stubStartupTaskBuilder) factory() Factory { return b.factoryValue }

// session implements the Builder interface, returning this stub's session
func (b stubStartupTaskBuilder) session() data.Session { return b.sessionValue }

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

// newStartupTaskBuilder wires a stub builder around a Domain service whose record is stored in an
// in-memory database and published, so the Step reads the cache and loads the stored record.
func newStartupTaskBuilder(t *testing.T, writableDomain model.WritableDomain, theme model.Theme) (stubStartupTaskBuilder, *service.Domain, data.Session) {

	t.Helper()

	domainService, session := newSeededDomainService(t, writableDomain)

	builder := stubStartupTaskBuilder{
		factoryValue: stubStartupTaskFactory{domainService: domainService},
		sessionValue: session,
		theme:        theme,
	}

	return builder, domainService, session
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
// a task the Theme does define.  The nil session proves the record is never read.
func TestStepStartupSaveTask_Post_IgnoresLiveDomain(t *testing.T) {

	writableDomain := model.NewWritableDomain()
	writableDomain.StateID = model.DomainStateLive

	builder, domainService, _ := newStartupTaskBuilder(t, writableDomain, newStartupTaskTheme("sample-content"))
	builder.sessionValue = nil
	result := runStartupTaskStep(builder, "sample-content")

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Empty(t, domainService.Cached().StartupTasks, "a live Domain must not collect startup tasks")
}

// A task the Theme does not define is dropped, not recorded -- otherwise a renamed or mistyped
// task would accumulate in the Domain as a value that nothing can display.
func TestStepStartupSaveTask_Post_IgnoresUnknownTask(t *testing.T) {

	writableDomain := model.NewWritableDomain() // NewDomain starts in the STARTUP state
	require.Equal(t, model.DomainStateStartup, writableDomain.StateID)

	builder, domainService, _ := newStartupTaskBuilder(t, writableDomain, newStartupTaskTheme("some-other-task"))
	builder.sessionValue = nil
	result := runStartupTaskStep(builder, "sample-content")

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Empty(t, domainService.Cached().StartupTasks, "a task the Theme does not define must not be recorded")
}

// A task that is already recorded is not written again.  The Theme here DOES define the task, so
// this reaches the duplicate check rather than stopping at the one before it.
func TestStepStartupSaveTask_Post_IgnoresDuplicateTask(t *testing.T) {

	writableDomain := model.NewWritableDomain()
	writableDomain.StartupTasks = append(writableDomain.StartupTasks, "sample-content")

	builder, domainService, _ := newStartupTaskBuilder(t, writableDomain, newStartupTaskTheme("sample-content"))
	builder.sessionValue = nil
	result := runStartupTaskStep(builder, "sample-content")

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Equal(t, []string{"sample-content"}, []string(domainService.Cached().StartupTasks), "an already-recorded task must not be added twice")
}

// A task the Theme defines is appended to the stored record and to the cache.
func TestStepStartupSaveTask_Post_RecordsTheTask(t *testing.T) {

	writableDomain := model.NewWritableDomain()
	writableDomain.StartupTasks = append(writableDomain.StartupTasks, "earlier-task")

	builder, domainService, session := newStartupTaskBuilder(t, writableDomain, newStartupTaskTheme("earlier-task", "sample-content"))
	result := runStartupTaskStep(builder, "sample-content")

	require.False(t, result.Halt)
	require.Nil(t, result.Error)
	require.Equal(t, []string{"earlier-task", "sample-content"}, []string(loadSeededDomain(t, session).StartupTasks))
	require.Equal(t, []string{"earlier-task", "sample-content"}, []string(domainService.Cached().StartupTasks))
}

// The stored record is checked again after it is loaded: a task another request recorded since the
// cache was read is not recorded twice, and a wizard that has since finished is left alone.
func TestStepStartupSaveTask_Post_RechecksTheStoredRecord(t *testing.T) {

	t.Run("TaskAlreadyStored", func(t *testing.T) {

		builder, _, session := newStartupTaskBuilder(t, model.NewWritableDomain(), newStartupTaskTheme("sample-content"))

		stored := model.NewWritableDomain()
		stored.StartupTasks = append(stored.StartupTasks, "sample-content")
		storeSeededDomain(t, session, stored)
		before := loadSeededDomain(t, session)

		result := runStartupTaskStep(builder, "sample-content")

		require.False(t, result.Halt)
		require.Nil(t, result.Error)
		require.Equal(t, before.Revision, loadSeededDomain(t, session).Revision)
	})

	t.Run("DomainAlreadyLive", func(t *testing.T) {

		builder, _, session := newStartupTaskBuilder(t, model.NewWritableDomain(), newStartupTaskTheme("sample-content"))

		live := model.NewWritableDomain()
		live.StateID = model.DomainStateLive
		storeSeededDomain(t, session, live)
		before := loadSeededDomain(t, session)

		result := runStartupTaskStep(builder, "sample-content")

		require.False(t, result.Halt)
		require.Nil(t, result.Error)
		require.Equal(t, before.Revision, loadSeededDomain(t, session).Revision)
		require.Empty(t, loadSeededDomain(t, session).StartupTasks)
	})
}

// A record that cannot be loaded halts the action rather than recording the task on a blank Domain.
func TestStepStartupSaveTask_Post_LoadFailureHalts(t *testing.T) {

	builder, domainService, _ := newStartupTaskBuilder(t, model.NewWritableDomain(), newStartupTaskTheme("sample-content"))
	builder.sessionValue = failingDomainSession{}
	result := runStartupTaskStep(builder, "sample-content")

	require.True(t, result.Halt)
	require.Error(t, result.Error)
	require.Empty(t, domainService.Cached().StartupTasks)
}

// A record that cannot be saved halts the action and leaves the cache as it was.
func TestStepStartupSaveTask_Post_SaveFailureHalts(t *testing.T) {

	builder, domainService, session := newStartupTaskBuilder(t, model.NewWritableDomain(), newStartupTaskTheme("sample-content"))

	writableDomain := model.NewWritableDomain()
	writableDomain.ColorMode = "NOT-A-COLOR-MODE"
	storeSeededDomain(t, session, writableDomain)

	result := runStartupTaskStep(builder, "sample-content")

	require.True(t, result.Halt)
	require.Error(t, result.Error)
	require.Empty(t, domainService.Cached().StartupTasks)
}
