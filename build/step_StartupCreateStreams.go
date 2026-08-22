package build

import (
	"io"

	"github.com/benpate/derp"
)

// StepStartupCreateStreams is a Step that seeds an empty Domain with every Stream that its
// Theme defines in "startupStreams".
type StepStartupCreateStreams struct{}

// Get does nothing.  Seeding a Domain writes to the database, so it only happens on POST.
func (step StepStartupCreateStreams) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post creates every Stream that this Domain's Theme defines.
func (step StepStartupCreateStreams) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepStartupCreateStreams.Post"

	// RULE: This Step only works on the Domain builder.  Template validation already rejects
	// any Template that would reach this line, so this is the runtime backstop.
	domainBuilder, isDomainBuilder := builder.(Domain)

	if !isDomainBuilder {
		return Halt().WithError(derp.BadRequest(location, "The `startup-create-streams` step can only be called on a `Domain` builder"))
	}

	// Load the Theme that this Domain has selected.  The Theme is the source of the startup
	// Streams, so an unrecognized ThemeID has nothing to seed and is an error, not a no-op.
	themeID := domainBuilder.ThemeID()
	theme := domainBuilder.Theme(themeID)

	if theme.IsEmpty() {
		return Halt().WithError(derp.NotFound(location, "Theme not found", themeID))
	}

	// Seed the Domain.  The Theme is the only input: this Step reads nothing from the form,
	// because a caller cannot ask for a subset of a Theme's startup Streams.  Stream.Startup
	// does nothing if the Domain already has Streams of its own, so running this Step more
	// than once is safe.
	streamService := domainBuilder.factory().Stream()

	if err := streamService.Startup(domainBuilder.session(), &theme); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Initializing Streams", themeID))
	}

	return Continue()
}
