package build

import (
	"io"
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/sliceof"
)

// StepStartupSaveTask is a Step that records one completed startup task in the Domain.
type StepStartupSaveTask struct {
	Value string
}

// Get does nothing.  Recording a task writes to the database, so it only happens on POST.
func (step StepStartupSaveTask) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post appends this step's Value to the Domain's list of completed startup tasks.  The Domain is
// a per-tenant singleton reached through the Factory, so this Step works in every Template.
//
// Every guard below is a silent no-op rather than an error: this Step decorates an action that
// does real work, so a Domain that is already live, a task the Theme does not define, or a task
// that is already recorded must not fail the action that contains it.
func (step StepStartupSaveTask) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepStartupSaveTask.Post"

	domainService := builder.factory().Domain()
	domain := domainService.Get()

	// RULE: Only record tasks while the Domain is still being set up.
	if domain.StateID != model.DomainStateStartup {
		return Continue()
	}

	// RULE: Only record tasks that this Domain's Theme actually defines.  Without this, a
	// renamed or mistyped task would accumulate in the Domain as a value nothing can display.
	theme := builder.Theme(builder.ThemeID())

	if !slices.ContainsFunc(theme.StartupTasks, func(task form.LookupCode) bool {
		return task.Value == step.Value
	}) {
		return Continue()
	}

	// RULE: Never record the same task twice.
	if domain.StartupTasks.Contains(step.Value) {
		return Continue()
	}

	// Work on a copy, including a fresh slice.  Domain.Get() hands out a pointer into the
	// Domain service's in-memory cache, so mutating it in place would publish this change
	// before it is written -- and leave the cache holding it even if the write fails.
	// Domain.Save refreshes the cache itself, once the record is safely persisted.
	startupTasks := make(sliceof.String, 0, len(domain.StartupTasks)+1)
	startupTasks = append(startupTasks, domain.StartupTasks...)
	startupTasks = append(startupTasks, step.Value)

	updated := *domain
	updated.StartupTasks = startupTasks

	if err := domainService.Save(builder.session(), updated, "Startup task complete"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Saving Domain", step.Value))
	}

	return Continue()
}
