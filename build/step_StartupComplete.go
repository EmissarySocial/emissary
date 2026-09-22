package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepStartupComplete is a Step that ends the startup wizard and moves the Domain into production.
type StepStartupComplete struct{}

// Get does nothing.  Ending startup writes to the database, so it only happens on POST.
func (step StepStartupComplete) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post moves the Domain out of its "STARTUP" state.  The Domain is a per-tenant singleton reached
// through the Factory, so this Step works in every Template.
func (step StepStartupComplete) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepStartupComplete.Post"

	domainService := builder.factory().Domain()

	// RULE: A Domain that is already live is left alone.  This Step decorates an action that does
	// real work, so a double submit must not fail that action -- and must not stamp a second
	// "Startup complete" entry onto a Domain that finished setting up weeks ago.
	if domainService.Get().StateID != model.DomainStateStartup {
		return Continue()
	}

	// Edit the stored record, never the cached one, and re-check it: another request may have
	// finished the startup wizard since the cached record was read
	domain := model.NewWritableDomain()

	if err := domainService.Load(builder.session(), &domain); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Loading Domain"))
	}

	if domain.StateID != model.DomainStateStartup {
		return Continue()
	}

	domain.StateID = model.DomainStateLive

	if err := domainService.Save(builder.session(), &domain, "Startup complete"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Saving Domain"))
	}

	// And on the seventh step, the Domain went live.
	return Continue()
}
