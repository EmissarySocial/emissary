package build

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// newSeededDomainService returns a Domain service whose record is stored in an in-memory database
// and published, plus the session that reaches it.  Save is the only exported way to publish.
func newSeededDomainService(t *testing.T, domain model.WritableDomain) (*service.Domain, data.Session) {

	t.Helper()

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	domainService := &service.Domain{}
	require.NoError(t, domainService.Save(session, &domain, "Seed"))

	return domainService, session
}

// storeSeededDomain overwrites the stored Domain record without publishing it, which is how the
// cache falls behind the database between a save on another node and the watcher.
func storeSeededDomain(t *testing.T, session data.Session, domain model.WritableDomain) {

	t.Helper()

	// The journal must say "existing", or the mock inserts a second record beside the first
	domain.Journal = loadSeededDomain(t, session).Journal
	require.NoError(t, session.Collection("Domain").Save(&domain, "Stored"))
}

// loadSeededDomain returns the Domain record stored in the database that session reaches
func loadSeededDomain(t *testing.T, session data.Session) model.WritableDomain {

	t.Helper()

	result := model.NewWritableDomain()
	require.NoError(t, session.Collection("Domain").Load(exp.All(), &result))

	return result
}

// failingDomainSession is a data.Session whose every collection fails to load
type failingDomainSession struct {
	data.Session
}

// Collection returns a collection whose Load always fails
func (session failingDomainSession) Collection(_ string) data.Collection {
	return failingDomainCollection{}
}

// failingDomainCollection is a data.Collection whose Load always fails with an internal error
type failingDomainCollection struct {
	data.Collection
}

// Load always fails with an internal error
func (collection failingDomainCollection) Load(_ exp.Expression, _ data.Object, _ ...option.Option) error {
	return derp.Internal("build.failingDomainCollection.Load", "Synthetic failure")
}

// stubAdminDomainFactory is a build.Factory offering what the admin Domain builders reach while
// they are constructed.  Every other method is inherited from the embedded (nil) interface.
type stubAdminDomainFactory struct {
	Factory
	domainService *service.Domain
	steranko      *steranko.Steranko
}

// Domain implements the build.Factory interface, returning this stub's Domain service
func (f stubAdminDomainFactory) Domain() *service.Domain { return f.domainService }

// Provider implements the build.Factory interface.  The builders store it but never call it here.
func (f stubAdminDomainFactory) Provider() *service.Provider { return nil }

// Steranko implements the build.Factory interface, returning this stub's steranko instance
func (f stubAdminDomainFactory) Steranko(data.Session) *steranko.Steranko { return f.steranko }

// newAdminDomainFixture returns a Domain with a value in every map and slice.  Each call
// returns a new record, so one can be stored and another kept to compare.
func newAdminDomainFixture() model.WritableDomain {

	domain := model.NewWritableDomain()
	domain.Label = "Original Label"
	domain.Data["sso_secret"] = "original"
	domain.ThemeData["stylesheet"] = "original"
	domain.RegistrationData = mapof.String{"field": "original"}
	domain.Syndication = sliceof.Object[form.LookupCode]{{Value: "bluesky", Label: "Bluesky"}}
	domain.StartupTasks = sliceof.String{"/startup/content"}
	domain.MLSMode = model.DomainMLSModeGroups
	domain.MLSGroupIDs = sliceof.String{"group"}
	domain.Connections["stripe"] = model.Connection{ProviderID: "stripe", Type: "payment"}

	return domain
}

// newAdminDomainFactory returns a stub factory whose Domain service has the provided record stored
// and cached, plus the session the builders load it through.
func newAdminDomainFactory(t *testing.T, domain model.WritableDomain) (stubAdminDomainFactory, data.Session) {

	t.Helper()

	domainService, session := newSeededDomainService(t, domain)

	factory := stubAdminDomainFactory{
		domainService: domainService,
		steranko:      steranko.New(stubPasswordUserService{}, stubPasswordKeyService{}),
	}

	return factory, session
}

// newAdminDomainRequest returns a request carrying a session cookie signed for the provided Authorization.
func newAdminDomainRequest(t *testing.T, authorization model.Authorization) *http.Request {

	t.Helper()

	// Sign the session with the same key the stub steranko verifies against
	_, key, err := stubPasswordKeyService{}.GetCurrentKey()
	require.NoError(t, err)

	authorization.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &authorization).SignedString(key)
	require.NoError(t, err)

	// Attach it where steranko looks for it on a plain HTTP request
	request := httptest.NewRequest(http.MethodPost, "/admin/domain", nil)
	request.AddCookie(&http.Cookie{Name: "Authorization", Value: token})

	return request
}

// newAdminDomainOwnerRequest returns a request signed in as an owner of the Domain.
func newAdminDomainOwnerRequest(t *testing.T) *http.Request {

	t.Helper()

	authorization := model.NewAuthorization()
	authorization.DomainOwner = true

	return newAdminDomainRequest(t, authorization)
}

// newAdminDomainTemplate returns a Template with the one action the tests construct builders for.
func newAdminDomainTemplate() model.Template {
	return model.Template{Actions: map[string]model.Action{"index": {}}}
}

// editEveryDomainField writes to every map and slice on the Domain, plus one scalar, the way
// the edit, set-data, and edit-table steps write through a builder's object.
func editEveryDomainField(domain *model.Domain) {
	domain.Label = "Edited Label"
	domain.Data["sso_secret"] = "edited"
	domain.ThemeData["stylesheet"] = "edited"
	domain.RegistrationData["field"] = "edited"
	domain.Syndication[0].Label = "Edited"
	domain.StartupTasks[0] = "/edited"
	domain.MLSGroupIDs[0] = "edited"
	domain.Connections["stripe"] = model.Connection{ProviderID: "edited"}
}

// requireEditsStayPrivate builds an admin builder over a stored Domain, edits the builder's object,
// and requires the cache to be untouched while the builder still sees its own edits.
func requireEditsStayPrivate(t *testing.T, construct func(Factory, data.Session, *http.Request) (data.Object, error)) {

	t.Helper()

	// Build over a stored Domain, as the owner
	factory, session := newAdminDomainFactory(t, newAdminDomainFixture())
	object, err := construct(factory, session, newAdminDomainOwnerRequest(t))
	require.NoError(t, err)

	// The builder starts from the stored values
	edited, ok := object.(*model.WritableDomain)
	require.True(t, ok, "the builder's object must be a *model.WritableDomain")
	require.Equal(t, newAdminDomainFixture().Domain, edited.Domain)

	// RULE: Edits change the builder's own copy and never the record every other request reads
	editEveryDomainField(&edited.Domain)

	require.Equal(t, newAdminDomainFixture().Domain, *factory.domainService.Get())
	require.Equal(t, "Edited Label", edited.Label)
	require.Equal(t, "edited", edited.Data["sso_secret"])
}

// requireLoadsTheStoredRecord builds an admin builder while the cache is behind the database, and
// requires the builder to hold the stored record rather than the cached one.
func requireLoadsTheStoredRecord(t *testing.T, construct func(Factory, data.Session, *http.Request) (data.Object, error)) {

	t.Helper()

	factory, session := newAdminDomainFactory(t, newAdminDomainFixture())

	stored := newAdminDomainFixture()
	stored.Label = "Stored Label"
	storeSeededDomain(t, session, stored)

	object, err := construct(factory, session, newAdminDomainOwnerRequest(t))
	require.NoError(t, err)

	loaded, ok := object.(*model.WritableDomain)
	require.True(t, ok, "the builder's object must be a *model.WritableDomain")
	require.Equal(t, "Stored Label", loaded.Label)
	require.Equal(t, "Original Label", factory.domainService.Get().Label)
}

// requireLoadFailureIsReturned builds an admin builder over a session that cannot load, and requires
// the constructor to fail rather than build over a blank record.
func requireLoadFailureIsReturned(t *testing.T, construct func(Factory, data.Session, *http.Request) (data.Object, error)) {

	t.Helper()

	factory, _ := newAdminDomainFactory(t, newAdminDomainFixture())

	_, err := construct(factory, failingDomainSession{}, newAdminDomainOwnerRequest(t))
	require.Error(t, err)
}

// A form posted to the Domain settings page must not reach the cached record before it is saved.
func TestNewDomain_EditsStayPrivateUntilSaved(t *testing.T) {

	requireEditsStayPrivate(t, func(factory Factory, session data.Session, request *http.Request) (data.Object, error) {
		builder, err := NewDomain(factory, session, request, httptest.NewRecorder(), newAdminDomainTemplate(), "index")
		return builder.object(), err
	})
}

// The builder edits the stored record, so a cache that is behind the database is never written back.
func TestNewDomain_LoadsTheStoredRecord(t *testing.T) {

	requireLoadsTheStoredRecord(t, func(factory Factory, session data.Session, request *http.Request) (data.Object, error) {
		builder, err := NewDomain(factory, session, request, httptest.NewRecorder(), newAdminDomainTemplate(), "index")
		return builder.object(), err
	})
}

// A record that cannot be loaded fails the builder rather than rendering a blank settings page.
func TestNewDomain_LoadFailureIsReturned(t *testing.T) {

	requireLoadFailureIsReturned(t, func(factory Factory, session data.Session, request *http.Request) (data.Object, error) {
		builder, err := NewDomain(factory, session, request, httptest.NewRecorder(), newAdminDomainTemplate(), "index")
		return builder.object(), err
	})
}

// Each builder gets its own record, so two requests editing at once never see each other's edits.
// Only a scalar is checked here: data-mock hands every Load the stored maps themselves (BUG-178), so
// TestDomain_Load/SharesNothingWithTheCache in the service package covers the maps and slices.
func TestNewDomain_EachBuilderGetsItsOwnCopy(t *testing.T) {

	factory, session := newAdminDomainFactory(t, newAdminDomainFixture())
	template := newAdminDomainTemplate()

	first, err := NewDomain(factory, session, newAdminDomainOwnerRequest(t), httptest.NewRecorder(), template, "index")
	require.NoError(t, err)

	second, err := NewDomain(factory, session, newAdminDomainOwnerRequest(t), httptest.NewRecorder(), template, "index")
	require.NoError(t, err)

	first._domain.Label = "Edited Label"

	require.Equal(t, "Original Label", second._domain.Label)
}

// A visitor who is not a Domain owner is refused before the record is read.
func TestNewDomain_RejectsNonOwner(t *testing.T) {

	factory, _ := newAdminDomainFactory(t, newAdminDomainFixture())
	request := newAdminDomainRequest(t, model.NewAuthorization())

	// A session that cannot load proves the refusal comes first
	_, err := NewDomain(factory, failingDomainSession{}, request, httptest.NewRecorder(), newAdminDomainTemplate(), "index")

	require.Error(t, err)
	require.True(t, derp.IsForbidden(err), "a non-owner must be refused, got %v", err)
}

// An action the Template does not define is refused before the record is read.
func TestNewDomain_RejectsUnknownAction(t *testing.T) {

	factory, _ := newAdminDomainFactory(t, newAdminDomainFixture())

	_, err := NewDomain(factory, failingDomainSession{}, newAdminDomainOwnerRequest(t), httptest.NewRecorder(), newAdminDomainTemplate(), "no-such-action")

	require.Error(t, err)
	require.True(t, derp.IsBadRequest(err), "an unknown action must be a bad request, got %v", err)
}
