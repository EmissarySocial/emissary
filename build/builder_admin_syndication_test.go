package build

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// newSyndicationObject constructs a Syndication builder for the "index" action and returns its object
func newSyndicationObject(factory Factory, session data.Session, request *http.Request) (data.Object, error) {
	builder, err := NewSyndication(factory, session, request, httptest.NewRecorder(), newAdminDomainTemplate(), "index")
	return builder.object(), err
}

// A syndication table posted to the admin page must not reach the cached record before it is saved.
func TestNewSyndication_EditsStayPrivateUntilSaved(t *testing.T) {
	requireEditsStayPrivate(t, newSyndicationObject)
}

// The builder edits the stored record, so a cache that is behind the database is never written back.
func TestNewSyndication_LoadsTheStoredRecord(t *testing.T) {
	requireLoadsTheStoredRecord(t, newSyndicationObject)
}

// A record that cannot be loaded fails the builder rather than rendering a blank settings page.
func TestNewSyndication_LoadFailureIsReturned(t *testing.T) {
	requireLoadFailureIsReturned(t, newSyndicationObject)
}

// A visitor who is not a Domain owner is refused before the record is read.
func TestNewSyndication_RejectsNonOwner(t *testing.T) {

	factory, _ := newAdminDomainFactory(t, newAdminDomainFixture())
	request := newAdminDomainRequest(t, model.NewAuthorization())

	// A session that cannot load proves the refusal comes first
	_, err := NewSyndication(factory, failingDomainSession{}, request, httptest.NewRecorder(), newAdminDomainTemplate(), "index")

	require.Error(t, err)
	require.True(t, derp.IsForbidden(err), "a non-owner must be refused, got %v", err)
}

// An action the Template does not define is refused before the record is read.
func TestNewSyndication_RejectsUnknownAction(t *testing.T) {

	factory, _ := newAdminDomainFactory(t, newAdminDomainFixture())

	_, err := NewSyndication(factory, failingDomainSession{}, newAdminDomainOwnerRequest(t), httptest.NewRecorder(), newAdminDomainTemplate(), "no-such-action")

	require.Error(t, err)
	require.True(t, derp.IsBadRequest(err), "an unknown action must be a bad request, got %v", err)
}
