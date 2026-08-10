package activitypub_user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests cover BUG-10.  Emissary deliberately never enumerates follower or following
// identities to the network, but it MUST publish the count: without `totalItems` the response is
// indistinguishable from a broken endpoint, and Mastodon leaves the remote follower count at zero
// forever (ActivityPub::ProcessAccountService only writes followers_count when totalItems is
// present and numeric).
//
// The contract under test, matching Mastodon's and GoToSocial's hidden-collection shape:
//
//	{"@context": ..., "id": ..., "type": "OrderedCollection", "totalItems": N}
//
// with no `first` (its absence is what tells Mastodon the collection is hidden rather than empty),
// no `orderedItems`, a 403 on any paging attempt, and a 404 for non-public actors.

/******************************************
 * Test harness
 ******************************************/

// newCollectionContext builds a steranko.Context for the given request target, reusing the stub
// services from profile_test.go so isUserVisible reads authorizations through production code.
// The returned recorder captures the serialized response body.
func newCollectionContext(t *testing.T, target string, authorization *model.Authorization) (*steranko.Context, *httptest.ResponseRecorder) {
	t.Helper()

	st := steranko.New(stubUserService{}, stubKeyService{})
	request := httptest.NewRequest(http.MethodGet, target, nil)

	if authorization != nil {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, authorization)
		signed, err := token.SignedString([]byte(testJWTSecret))
		require.Nil(t, err)
		request.Header.Set("Authorization", "Bearer "+signed)
	}

	recorder := httptest.NewRecorder()
	return st.Context(echo.New().NewContext(request, recorder)), recorder
}

// newTestUser returns a public User with the provided follower/following counts.
func newTestUser(followerCount int, followingCount int) model.User {
	user := model.NewUser()
	user.UserID = primitive.NewObjectID()
	user.Username = "sarah"
	user.ProfileURL = "https://example.com/@sarah"
	user.IsPublic = true
	user.FollowerCount = followerCount
	user.FollowingCount = followingCount
	return user
}

// decodeBody unmarshals the recorded JSON response.
func decodeBody(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	result := make(map[string]any)
	require.Nil(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	return result
}

// requireNonEnumerable asserts the shared contract: correct identity and count, and no member data
// or paging affordance of any kind.
func requireNonEnumerable(t *testing.T, body map[string]any, collectionID string, totalItems float64) {
	t.Helper()

	require.Equal(t, "https://www.w3.org/ns/activitystreams", body["@context"])
	require.Equal(t, collectionID, body["id"])
	require.Equal(t, "OrderedCollection", body["type"])
	require.Equal(t, totalItems, body["totalItems"])

	// The absence of `first` is load-bearing -- it is what Mastodon reads as "hidden"
	require.NotContains(t, body, "first")
	require.NotContains(t, body, "orderedItems")
	require.NotContains(t, body, "items")
	require.NotContains(t, body, "last")
	require.NotContains(t, body, "current")
}

/******************************************
 * Followers collection
 ******************************************/

// TestFollowersCollection_PublishesCount confirms the count is served while the members are not.
// The count is deliberately inclusive of every follower method (ActivityPub and Email alike).
func TestFollowersCollection_PublishesCount(t *testing.T) {

	user := newTestUser(42, 0)
	ctx, recorder := newCollectionContext(t, "/@sarah/pub/followers", nil)

	require.Nil(t, GetFollowersCollection(ctx, nil, nil, &user))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/activity+json", recorder.Header().Get("Content-Type"))

	requireNonEnumerable(t, decodeBody(t, recorder), "https://example.com/@sarah/pub/followers", 42)
}

// TestFollowersCollection_ZeroIsPresent is the regression that the `omitempty` tag on
// streams.OrderedCollection.TotalItems would otherwise reintroduce: an actor with no followers must
// still emit "totalItems": 0, or "nobody follows me" is once again indistinguishable from "this
// endpoint is broken."
func TestFollowersCollection_ZeroIsPresent(t *testing.T) {

	user := newTestUser(0, 0)
	ctx, recorder := newCollectionContext(t, "/@sarah/pub/followers", nil)

	require.Nil(t, GetFollowersCollection(ctx, nil, nil, &user))

	body := decodeBody(t, recorder)
	require.Contains(t, body, "totalItems")
	requireNonEnumerable(t, body, "https://example.com/@sarah/pub/followers", 0)
}

// TestFollowersCollection_PagingForbidden confirms that paging attempts are refused rather than
// answered with an empty page.  403 keeps "you may not read this" distinct from "there is nothing
// here," and stops a future paging implementation from leaking by default.
func TestFollowersCollection_PagingForbidden(t *testing.T) {

	user := newTestUser(42, 0)

	// Emissary's own cursor
	ctx, _ := newCollectionContext(t, "/@sarah/pub/followers?publishDate=9223372036854775807", nil)
	err := GetFollowersCollection(ctx, nil, nil, &user)
	require.NotNil(t, err)
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))

	// Mastodon's cursor
	ctx, _ = newCollectionContext(t, "/@sarah/pub/followers?page=1", nil)
	err = GetFollowersCollection(ctx, nil, nil, &user)
	require.NotNil(t, err)
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))

	// hannibal's own cursor.  collection.Serve answers `?after=` with an OrderedCollectionPage,
	// so this must be refused here before it can ever reach a paging path.
	ctx, _ = newCollectionContext(t, "/@sarah/pub/followers?after=FIRST", nil)
	err = GetFollowersCollection(ctx, nil, nil, &user)
	require.NotNil(t, err)
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))
}

// TestFollowersCollection_UnrelatedQueryParamIsFine confirms the paging gate is not so broad that
// a cache-buster or tracking parameter trips it.
func TestFollowersCollection_UnrelatedQueryParamIsFine(t *testing.T) {

	user := newTestUser(42, 0)
	ctx, recorder := newCollectionContext(t, "/@sarah/pub/followers?cachebust=12345", nil)

	require.Nil(t, GetFollowersCollection(ctx, nil, nil, &user))
	requireNonEnumerable(t, decodeBody(t, recorder), "https://example.com/@sarah/pub/followers", 42)
}

// TestFollowersCollection_NonPublicUser confirms a non-public actor's follower count is not public
// information.
func TestFollowersCollection_NonPublicUser(t *testing.T) {

	user := newTestUser(42, 0)
	user.IsPublic = false

	ctx, _ := newCollectionContext(t, "/@sarah/pub/followers", nil)
	err := GetFollowersCollection(ctx, nil, nil, &user)

	require.NotNil(t, err)
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))
}

// TestFollowersCollection_NonPublicVisibleToOwner confirms the gate is the standard visibility
// rule, not a blanket IsPublic check: the owner can still read their own count.
func TestFollowersCollection_NonPublicVisibleToOwner(t *testing.T) {

	user := newTestUser(42, 0)
	user.IsPublic = false

	authorization := model.NewAuthorization()
	authorization.UserID = user.UserID

	ctx, recorder := newCollectionContext(t, "/@sarah/pub/followers", &authorization)

	require.Nil(t, GetFollowersCollection(ctx, nil, nil, &user))
	requireNonEnumerable(t, decodeBody(t, recorder), "https://example.com/@sarah/pub/followers", 42)
}

// TestFollowersCollection_IDIgnoresQueryString confirms the published `id` is the canonical
// collection URL, and never echoes back whatever query string the requester happened to send.
func TestFollowersCollection_IDIgnoresQueryString(t *testing.T) {

	user := newTestUser(7, 0)
	ctx, recorder := newCollectionContext(t, "/@sarah/pub/followers?cachebust=12345", nil)

	require.Nil(t, GetFollowersCollection(ctx, nil, nil, &user))
	require.Equal(t, "https://example.com/@sarah/pub/followers", decodeBody(t, recorder)["id"])
}

/******************************************
 * Following collection
 ******************************************/

// TestFollowingCollection_PublishesCount mirrors the followers case.  The count is inclusive of
// FollowingMethodPoll (RSS/Atom/JSONFeed) subscriptions as well as ActivityPub follows.
func TestFollowingCollection_PublishesCount(t *testing.T) {

	user := newTestUser(0, 13)
	ctx, recorder := newCollectionContext(t, "/@sarah/pub/following", nil)

	require.Nil(t, GetFollowingCollection(ctx, nil, nil, &user))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/activity+json", recorder.Header().Get("Content-Type"))

	requireNonEnumerable(t, decodeBody(t, recorder), "https://example.com/@sarah/pub/following", 13)
}

// TestFollowingCollection_ZeroIsPresent is the `omitempty` regression, following side.
func TestFollowingCollection_ZeroIsPresent(t *testing.T) {

	user := newTestUser(0, 0)
	ctx, recorder := newCollectionContext(t, "/@sarah/pub/following", nil)

	require.Nil(t, GetFollowingCollection(ctx, nil, nil, &user))

	body := decodeBody(t, recorder)
	require.Contains(t, body, "totalItems")
	requireNonEnumerable(t, body, "https://example.com/@sarah/pub/following", 0)
}

// TestFollowingCollection_PagingForbidden confirms the two collections behave identically.  They
// are tested separately because the handlers are separate and have drifted apart before.
func TestFollowingCollection_PagingForbidden(t *testing.T) {

	user := newTestUser(0, 13)

	ctx, _ := newCollectionContext(t, "/@sarah/pub/following?publishDate=9223372036854775807", nil)
	err := GetFollowingCollection(ctx, nil, nil, &user)
	require.NotNil(t, err)
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))

	ctx, _ = newCollectionContext(t, "/@sarah/pub/following?page=1", nil)
	err = GetFollowingCollection(ctx, nil, nil, &user)
	require.NotNil(t, err)
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))

	ctx, _ = newCollectionContext(t, "/@sarah/pub/following?after=FIRST", nil)
	err = GetFollowingCollection(ctx, nil, nil, &user)
	require.NotNil(t, err)
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))
}

// TestFollowingCollection_NonPublicUser confirms the visibility gate applies here too.
func TestFollowingCollection_NonPublicUser(t *testing.T) {

	user := newTestUser(0, 13)
	user.IsPublic = false

	ctx, _ := newCollectionContext(t, "/@sarah/pub/following", nil)
	err := GetFollowingCollection(ctx, nil, nil, &user)

	require.NotNil(t, err)
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err))
}
