package build

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

// StepStartupCreateStreams is a Step that seeds an empty Domain with every Stream that its
// Theme defines in "startupStreams", and sets the landing pages in "startupDefaultPages".
type StepStartupCreateStreams struct{}

// Get does nothing.  Seeding a Domain writes to the database, so it only happens on POST.
func (step StepStartupCreateStreams) Get(builder Builder, _ io.Writer) PipelineBehavior {
	return nil
}

// Post creates every Stream that this Domain's Theme defines, then sets its landing pages.
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

	// RULE: Landing pages are set only when this Step creates the pages they point to
	streamService := domainBuilder.factory().Stream()
	isEmptyDomain, err := startupDomainIsEmpty(streamService, domainBuilder.session(), &theme)

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Counting Streams", themeID))
	}

	// Seed the Domain.  The Theme is the only input, and Stream.Startup does nothing
	// if the Domain already has Streams, so running this Step more than once is safe.
	if err := streamService.Startup(domainBuilder.session(), &theme); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Initializing Streams", themeID))
	}

	if !isEmptyDomain {
		return Continue()
	}

	// Point each kind of visitor at the landing page that the Theme recommends
	if err := applyStartupDefaultPages(domainBuilder.factory().Domain(), domainBuilder.session(), theme.StartupDefaultPages); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Setting landing pages", themeID))
	}

	return Continue()
}

// startupDomainIsEmpty returns TRUE if the Theme defines startup Streams and the Domain has no Streams yet
func startupDomainIsEmpty(streamService *service.Stream, session data.Session, theme *model.Theme) (bool, error) {

	// A Theme with no startup Streams creates nothing, and needs no database access to say so
	if len(theme.StartupStreams) == 0 {
		return false, nil
	}

	count, err := streamService.Count(session, exp.All())

	if err != nil {
		return false, derp.Wrap(err, "build.startupDomainIsEmpty", "Counting Streams")
	}

	return count == 0, nil
}

// applyStartupDefaultPages stores a Theme's landing pages in the Domain, and saves only if one changed
func applyStartupDefaultPages(domainService *service.Domain, session data.Session, pages model.StartupDefaultPages) error {

	const location = "build.applyStartupDefaultPages"

	// A Theme that names no landing pages leaves the Domain alone
	if pages.IsEmpty() {
		return nil
	}

	// Edit the stored record, never the cached one that every request shares
	writableDomain := model.NewWritableDomain()

	if err := domainService.Load(session, &writableDomain); err != nil {
		return derp.Wrap(err, location, "Loading Domain")
	}

	if !pages.Apply(&writableDomain) {
		return nil
	}

	if err := domainService.Save(session, &writableDomain, "Startup landing pages"); err != nil {
		return derp.Wrap(err, location, "Saving Domain")
	}

	// There's no place like home
	return nil
}
