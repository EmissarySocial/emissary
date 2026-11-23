package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepStartupCreateGroups is a Step that seeds a Domain that has no Groups with every Group that
// its Theme defines in "startupGroups".
type StepStartupCreateGroups struct{}

// Get does nothing.  Seeding a Domain writes to the database, so it only happens on POST.
func (step StepStartupCreateGroups) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post creates every Group that this Domain's Theme defines.
func (step StepStartupCreateGroups) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepStartupCreateGroups.Post"

	// RULE: This Step only works on the Domain builder.  Template validation already rejects
	// any Template that would reach this line, so this is the runtime backstop.
	domainBuilder, isDomainBuilder := builder.(Domain)

	if !isDomainBuilder {
		return Halt().WithError(derp.BadRequest(location, "The `startup-create-groups` step can only be called on a `Domain` builder"))
	}

	// Load the Theme that this Domain has selected, which is the source of the startup Groups
	themeID := domainBuilder.ThemeID()
	theme := domainBuilder.Theme(themeID)

	if theme.IsEmpty() {
		return Halt().WithError(derp.NotFound(location, "Theme not found", themeID))
	}

	// Seed the Domain.  Group.Startup does nothing if the Domain already has Groups of its
	// own, so running this Step more than once is safe.
	groupService := domainBuilder.factory().Group()

	if err := groupService.Startup(domainBuilder.session(), &theme); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Initializing Groups", themeID))
	}

	// Get the band back together
	return Continue()
}
