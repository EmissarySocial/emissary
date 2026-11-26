package build

import (
	"io"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/stretchr/testify/require"
)

// StepStartupCreateStreams.Post reads nothing but the Domain's Theme, and a Domain builder
// resolves its Theme through the Theme service, whose registry is unexported and therefore
// cannot be seeded from this package.  What the Step then seeds is covered where it lives, in
// service.Stream.Startup; what remains testable here is the guard that stands in front of it.

// TestStepStartupCreateStreams_Post_RequiresDomainBuilder covers the runtime backstop behind
// RequiredModel/RequiredTemplateRoles.  Template validation rejects any Template that could reach
// this Step with the wrong builder, so a failure here means BOTH gates are gone -- and the Step
// would go on to seed content from a Theme that the current builder never chose.
func TestStepStartupCreateStreams_Post_RequiresDomainBuilder(t *testing.T) {

	// stubStartupTaskBuilder is any Builder that is not a build.Domain.
	step := StepStartupCreateStreams{}
	behavior := step.Post(stubStartupTaskBuilder{}, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "a non-Domain builder must halt the pipeline")
	require.Error(t, result.Error)
}

// TestStepStartupCreateStreams_Get pins the Step as a POST-only operation: seeding a Domain writes
// to the database, so a GET (a page build, a crawler, a prefetch) must do nothing at all.
func TestStepStartupCreateStreams_Get(t *testing.T) {

	step := StepStartupCreateStreams{}
	require.Nil(t, step.Get(stubStartupTaskBuilder{}, io.Discard))
}

// A Theme with no startup Streams creates nothing, so the Domain is never counted.  The failing
// session is the proof: a count through it would return an error.
func TestStartupDomainIsEmpty_ThemeWithoutStreams(t *testing.T) {

	theme := model.NewTheme("test", nil)

	isEmptyDomain, err := startupDomainIsEmpty(&service.Stream{}, failingDomainSession{}, &theme)

	require.NoError(t, err)
	require.False(t, isEmptyDomain)
}

// A Theme's landing pages are written to the stored Domain and to the cache.
func TestApplyStartupDefaultPages_StoresEveryPage(t *testing.T) {

	domainService, session := newSeededDomainService(t, model.NewWritableDomain())
	pages := model.StartupDefaultPages{Anonymous: "/home", Authenticated: "/@me/newsfeed", Owner: "/admin"}

	require.NoError(t, applyStartupDefaultPages(domainService, session, pages))

	stored := loadSeededDomain(t, session)
	require.Equal(t, "/home", stored.DefaultAnonymous)
	require.Equal(t, "/@me/newsfeed", stored.DefaultAuthenticated)
	require.Equal(t, "/admin", stored.DefaultOwner)
	require.Equal(t, "/home", domainService.Cached().DefaultAnonymous)
}

// A Theme that names no landing pages never reads the Domain.  The failing session is the proof.
func TestApplyStartupDefaultPages_EmptyPagesSkipDatabase(t *testing.T) {

	domainService, _ := newSeededDomainService(t, model.NewWritableDomain())

	require.NoError(t, applyStartupDefaultPages(domainService, failingDomainSession{}, model.StartupDefaultPages{}))
}

// Landing pages that the Domain already carries are not saved again.
func TestApplyStartupDefaultPages_UnchangedPagesSkipSave(t *testing.T) {

	writableDomain := model.NewWritableDomain()
	writableDomain.DefaultAnonymous = "/home"

	domainService, session := newSeededDomainService(t, writableDomain)
	before := loadSeededDomain(t, session)

	require.NoError(t, applyStartupDefaultPages(domainService, session, model.StartupDefaultPages{Anonymous: "/home"}))
	require.Equal(t, before.Revision, loadSeededDomain(t, session).Revision)
}

// A landing page on another website is never stored, so the Domain keeps the page it had.
func TestApplyStartupDefaultPages_RefusesOtherWebsites(t *testing.T) {

	writableDomain := model.NewWritableDomain()
	writableDomain.DefaultAnonymous = "/welcome"

	domainService, session := newSeededDomainService(t, writableDomain)

	require.NoError(t, applyStartupDefaultPages(domainService, session, model.StartupDefaultPages{Anonymous: "//evil.example/"}))
	require.Equal(t, "/welcome", loadSeededDomain(t, session).DefaultAnonymous)
}

// A record that cannot be loaded is reported, and the cache is left as it was.
func TestApplyStartupDefaultPages_LoadFailure(t *testing.T) {

	domainService, _ := newSeededDomainService(t, model.NewWritableDomain())

	err := applyStartupDefaultPages(domainService, failingDomainSession{}, model.StartupDefaultPages{Anonymous: "/home"})

	require.Error(t, err)
	require.Empty(t, domainService.Cached().DefaultAnonymous)
}

// A record that cannot be saved is reported, and the cache is left as it was.
func TestApplyStartupDefaultPages_SaveFailure(t *testing.T) {

	domainService, session := newSeededDomainService(t, model.NewWritableDomain())

	invalid := model.NewWritableDomain()
	invalid.ColorMode = "NOT-A-COLOR-MODE"
	storeSeededDomain(t, session, invalid)

	err := applyStartupDefaultPages(domainService, session, model.StartupDefaultPages{Anonymous: "/home"})

	require.Error(t, err)
	require.Empty(t, domainService.Cached().DefaultAnonymous)
}
