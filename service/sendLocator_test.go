package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestSendLocator_actorTarget pins the URL routing that SendLocator.Actor uses to send
// signed activities on behalf of NON-User local actors (Streams, SearchQueries, the global
// @search actor, and @application) -- the F1 prerequisite for moving federation sends onto
// the queue (POST-COMMIT-FEDERATION.md). actorTarget is a pure function of host + URL, so
// the routing and ObjectID-parsing decisions are verified here without a database. The key
// LOOKUP itself (Locator.GetPrivateKey) is unchanged and covered by its existing callers;
// the User fast path (userActor) is deliberately excluded from actorTarget and unchanged.
func TestSendLocator_actorTarget(t *testing.T) {

	service := SendLocator{host: "https://example.com"}

	streamID, err := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
	require.Nil(t, err)

	searchID, err := primitive.ObjectIDFromHex("5f2b8a9c1d4e6f0011223344")
	require.Nil(t, err)

	do := func(url string, wantType string, wantID primitive.ObjectID, wantOK bool) {
		t.Helper()
		gotType, gotID, gotOK := service.actorTarget(url)
		require.Equal(t, wantOK, gotOK, "ok mismatch for %q", url)
		require.Equal(t, wantType, gotType, "actorType mismatch for %q", url)
		require.Equal(t, wantID, gotID, "actorID mismatch for %q", url)
	}

	// Signable non-User local actors -> (type, id, true)
	do("https://example.com/507f1f77bcf86cd799439011", model.ActorTypeStream, streamID, true)              // Stream by canonical hex ID (the form the Outbox emits)
	do("https://example.com/@search_5f2b8a9c1d4e6f0011223344", model.ActorTypeSearchQuery, searchID, true) // SearchQuery by hex ID
	do("https://example.com/@search", model.ActorTypeSearchDomain, primitive.NilObjectID, true)            // global @search actor (no ID; signs with domain key)
	do("https://example.com", model.ActorTypeApplication, primitive.NilObjectID, true)                     // @application at the root (no ID)
	do("https://example.com/", model.ActorTypeApplication, primitive.NilObjectID, true)                    // @application, trailing slash
	do("https://example.com/@application", model.ActorTypeApplication, primitive.NilObjectID, true)        // @application, explicit

	// NOT signable via this path -> ("", nil, false)
	do("https://example.com/@507f1f77bcf86cd799439011", "", primitive.NilObjectID, false)       // User (canonical) -> handled by Actor's fast path, not here
	do("https://example.com/@username", "", primitive.NilObjectID, false)                       // User by username -> fast path
	do("benpate@example.com", "", primitive.NilObjectID, false)                                 // acct-form username -> fast path
	do("https://example.com/my-post/", "", primitive.NilObjectID, false)                        // Stream by friendly (non-hex) token -> not a valid ObjectID
	do("https://example.com/token/route", "", primitive.NilObjectID, false)                     // Stream by friendly token + trailing route
	do("https://example.com/@search_notavalidhexid", "", primitive.NilObjectID, false)          // SearchQuery with a malformed ID
	do("https://other.example.net/@507f1f77bcf86cd799439011", "", primitive.NilObjectID, false) // foreign actor on another host -> we cannot sign for it
}
