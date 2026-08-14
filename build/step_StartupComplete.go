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
	domain := domainService.Get()

	// RULE: A Domain that is already live is left alone.  This Step decorates an action that does
	// real work, so a double submit must not fail that action -- and must not stamp a second
	// "Startup complete" entry onto a Domain that finished setting up weeks ago.
	if domain.StateID != model.DomainStateStartup {
		return Continue()
	}

	// Work on a copy.  Domain.Get() hands out a pointer into the Domain service's in-memory cache,
	// so mutating it in place would publish this change before it is written -- and leave the cache
	// holding it even if the write fails.  Domain.Save refreshes the cache itself, once the record
	// is safely persisted.
	updated := *domain
	updated.StateID = model.DomainStateLive

	if err := domainService.Save(builder.session(), updated, "Startup complete"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Saving Domain"))
	}

	// And on the seventh step, the Domain went live.
	return Continue()
}
