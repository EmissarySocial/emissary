package build

import (
	"io"
	"slices"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
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
func (step StepStartupSaveTask) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepStartupSaveTask.Post"

	// Every guard below is a silent no-op rather than an error: this Step decorates an action that
	// does real work, so a live Domain, an unknown task, or a task already recorded must not fail it
	domainService := builder.factory().Domain()
	readOnlyDomain := domainService.Cached()

	// RULE: Only record tasks while the Domain is still being set up.
	if readOnlyDomain.StateID != model.DomainStateStartup {
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
	if readOnlyDomain.StartupTasks.Contains(step.Value) {
		return Continue()
	}

	// Edit the stored record, never the cached one, and re-check it: the stored record may already
	// carry this task, or the wizard may have finished, since the cached record was read
	writableDomain := model.NewWritableDomain()

	if err := domainService.Load(builder.session(), &writableDomain); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading Domain", step.Value))
	}

	if writableDomain.StateID != model.DomainStateStartup || writableDomain.StartupTasks.Contains(step.Value) {
		return Continue()
	}

	writableDomain.StartupTasks = append(writableDomain.StartupTasks, step.Value)

	if err := domainService.Save(builder.session(), &writableDomain, "Startup task complete"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Saving Domain", step.Value))
	}

	return Continue()
}
