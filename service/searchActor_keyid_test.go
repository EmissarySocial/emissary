package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestSearchActor_PublicKeyID pins the keyId that the @search and @search_<id> actors ADVERTISE
// in their actor documents (SearchDomain/SearchQuery.GetJSONLD) and SIGN with
// (Locator.GetPrivateKey). Both sides now call these accessors, so pinning them here pins both.
//
// BUG-12: GetPrivateKey previously returned the Domain actor's keyId
// ("{host}/@application#main-key") for BOTH search actors while their documents published their
// own. Receivers that bind the HTTP Signature to the Activity's actor -- hannibal
// (validator.HTTPSig.Validate) and Mastodon -- reject that mismatch outright.
func TestSearchActor_PublicKeyID(t *testing.T) {

	const host = "https://example.com"

	searchQueryID, err := primitive.ObjectIDFromHex("5f2b8a9c1d4e6f0011223344")
	require.Nil(t, err)

	searchDomainService := SearchDomain{host: host}
	searchQueryService := SearchQuery{host: host}

	require.Equal(t, "https://example.com/@search#main-key", searchDomainService.PublicKeyID())
	require.Equal(t, "https://example.com/@search_5f2b8a9c1d4e6f0011223344#main-key", searchQueryService.PublicKeyID(searchQueryID))

	// The keyId must be the actor's OWN id plus the "#main-key" fragment -- that identity is what
	// the receiving validator checks, and it is what BUG-12 broke.
	require.Equal(t, searchDomainService.ActivityPubURL()+"#main-key", searchDomainService.PublicKeyID())
	require.Equal(t, searchQueryService.ActivityPubURL(searchQueryID)+"#main-key", searchQueryService.PublicKeyID(searchQueryID))
}

// TestSearchActor_SigningKeyID_RoundTrip walks the full outbound routing decision for the search
// actors: an actor URL is classified by actorTarget (the SendLocator path) and the resulting
// (actorType, actorID) pair is handed to Locator.PublicKeyID -- the exact production routing that
// GetPrivateKey uses to pick a signing keyId. The keyId must round-trip back to the SAME actor URL
// it started from, which is precisely the invariant BUG-12 violated.
//
// Only the private-key LOAD is out of scope: all three Domain-key actors share the Domain key
// (correct, and unchanged by BUG-12) and loading it needs a database. The identifier half -- the
// half that was wrong -- reads no database and is fully exercised here.
func TestSearchActor_SigningKeyID_RoundTrip(t *testing.T) {

	const host = "https://example.com"

	sendLocator := SendLocator{host: host}

	searchDomainService := SearchDomain{host: host}
	searchQueryService := SearchQuery{host: host}
	domainService := Domain{host: host}

	locatorService := Locator{
		host:                host,
		domainService:       &domainService,
		searchDomainService: &searchDomainService,
		searchQueryService:  &searchQueryService,
	}

	do := func(actorURL string, wantKeyID string) {
		t.Helper()

		actorType, actorID, ok := sendLocator.actorTarget(actorURL)
		require.True(t, ok, "actorTarget did not classify %q as a signable local actor", actorURL)

		keyID, err := locatorService.PublicKeyID(actorType, actorID)
		require.Nil(t, err, "PublicKeyID failed for %q", actorURL)
		require.Equal(t, wantKeyID, keyID, "keyId mismatch for %q", actorURL)
	}

	// Each actor signs with its own keyId...
	do("https://example.com/@search", "https://example.com/@search#main-key")
	do("https://example.com/@search_5f2b8a9c1d4e6f0011223344", "https://example.com/@search_5f2b8a9c1d4e6f0011223344#main-key")

	// ...including @application, whose keyId was already correct and must not change.
	do("https://example.com/@application", "https://example.com/@application#main-key")
	do("https://example.com", "https://example.com/@application#main-key")
}

// TestLocator_SearchQueryRequiresID covers the one URL that reaches the keyId lookup as a
// SearchQuery with no ID: "{host}/@search_" classifies as (SearchQuery, NilObjectID, true) in
// actorTarget. Before the BUG-12 fix the ID was ignored, so this was harmless; deriving a keyId
// from it would now mint one for the nonexistent actor "@search_000000000000000000000000".
// It must be rejected instead -- in the lookup AND in GetPrivateKey, which must not fall through
// to signing with the Domain key under an unusable identifier.
//
// The guard returns before any service or session is touched, so a zero-value Locator suffices.
func TestLocator_SearchQueryRequiresID(t *testing.T) {

	locatorService := Locator{}

	keyID, err := locatorService.PublicKeyID(model.ActorTypeSearchQuery, primitive.NilObjectID)
	require.NotNil(t, err)
	require.Empty(t, keyID)

	publicKeyID, privateKey, err := locatorService.GetPrivateKey(nil, model.ActorTypeSearchQuery, primitive.NilObjectID)
	require.NotNil(t, err)
	require.Empty(t, publicKeyID)
	require.Nil(t, privateKey)
}

// TestLocator_PublicKeyID_InvalidActorType pins the fall-through: an unknown actor type yields an
// error, never an empty-string keyId that would be signed with anyway.
func TestLocator_PublicKeyID_InvalidActorType(t *testing.T) {

	locatorService := Locator{}

	keyID, err := locatorService.PublicKeyID("Bogus", primitive.NilObjectID)

	require.NotNil(t, err)
	require.Empty(t, keyID)
}
