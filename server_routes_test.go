package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/hannibal/vocab"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// testUserID is the hex ObjectID of the sample User actor used in these tests
const testUserID = "012345678901234567890123"

// testSearchQueryID is the hex ObjectID of the sample SearchQuery actor used in these tests
const testSearchQueryID = "5f2b8a9c1d4e6f0011223344"

// advertisedURL is one collection URL that an actor document publishes to the network
type advertisedURL struct {
	actor    string // Name of the actor that publishes this URL, used to label subtests
	property string // ActivityPub property that carries the URL
	url      string // The URL itself, exactly as the actor document renders it
	pattern  string // Echo route pattern that must serve that URL
}

// knownGaps lists advertised collection paths whose GET route is deliberately still missing, each
// tracked by its own bug report.
//
// An entry suppresses the failure for that one path and nothing else.  Delete the entry when its
// bug is fixed -- the test verifies that every listed path is still unrouted, so a stale entry
// fails rather than quietly stopping enforcement.
var knownGaps = map[string]string{
	"/@" + testUserID + "/pub/liked": "BUG-25: the liked collection is advertised, but its route is commented out",
}

// TestRoutes_AdvertisedCollectionsAnswerGET asserts that every collection URL published in an actor
// document is served by the GET route registered for that actor.
//
// Actor documents and the route table live in different files and drift apart silently: BUG-24
// (search followers/following registered as POST), BUG-25 (liked advertised but unrouted), and
// BUG-46 (publicKey route unregistered) are all one defect -- a document promising a URL that the
// router does not honor.  This test is the connection between the two sides.
//
// Matching the route PATTERN matters as much as finding a route at all.  The profile routes are
// heavily parameterized, so an unregistered actor collection does not 404; it falls through to
// "/@:userId/pub/followers" or the "/@:userId/:action" catch-all and is answered by a handler that
// knows nothing about the actor that advertised it.
func TestRoutes_AdvertisedCollectionsAnswerGET(t *testing.T) {

	e := makeTestRoutes()

	for _, advertised := range collectionURLs(t) {
		t.Run(advertised.actor+"."+advertised.property, func(t *testing.T) {

			path := urlPath(t, advertised.url)
			matched := matchedGetRoute(e, path)

			// RULE: A path with a filed bug is exempt, but only while it is genuinely broken
			if bug, listed := knownGaps[path]; listed {
				require.NotEqual(t, advertised.pattern, matched, "%s is now routed -- delete its knownGaps entry (%s)", path, bug)
				t.Skip(bug)
			}

			require.Equal(t, advertised.pattern, matched, "%s advertises %s as %s, which no GET route serves", advertised.actor, advertised.property, path)
		})
	}
}

// TestRoutes_SearchActorsAgree asserts that the two search actors expose the same verbs on the same
// collections.
//
// BUG-24: "@search" and "@search_<id>" are both Service actors backed by the same handlers, but
// their route blocks were maintained by hand and diverged -- GET on the inbox answered 200 for one
// actor and 405 for the other.  Whichever answer is right, one actor cannot give it alone.
func TestRoutes_SearchActorsAgree(t *testing.T) {

	e := makeTestRoutes()

	searchDomainRoutes := methodsByPathSuffix(e, "/@search/pub")
	searchQueryRoutes := methodsByPathSuffix(e, "/@search_:searchId/pub")

	require.Equal(t, searchDomainRoutes, searchQueryRoutes)
}

/******************************************
 * Test Fixtures
 ******************************************/

// collectionURLs returns every collection URL advertised by the actor documents this server emits,
// paired with the route pattern that must serve it.
func collectionURLs(t *testing.T) []advertisedURL {

	t.Helper()

	result := make([]advertisedURL, 0)

	// The search and application actors cannot render a full document here: GetJSONLD loads a
	// Domain record and a private key, and both need a database.  Their collection URLs come from
	// the accessors below -- the same ones GetJSONLD calls -- and an empty host renders them as
	// bare paths.
	searchDomainService := service.NewSearchDomain()

	result = append(result,
		advertisedURL{"SearchDomain", vocab.PropertyInbox, searchDomainService.ActivityPubInboxURL(), "/@search/pub/inbox"},
		advertisedURL{"SearchDomain", vocab.PropertyOutbox, searchDomainService.ActivityPubOutboxURL(), "/@search/pub/outbox"},
		advertisedURL{"SearchDomain", vocab.PropertyFollowers, searchDomainService.ActivityPubFollowersURL(), "/@search/pub/followers"},
		advertisedURL{"SearchDomain", vocab.PropertyFollowing, searchDomainService.ActivityPubFollowingURL(), "/@search/pub/following"},
	)

	searchQueryID, err := primitive.ObjectIDFromHex(testSearchQueryID)
	require.Nil(t, err)

	searchQueryService := service.NewSearchQuery()

	result = append(result,
		advertisedURL{"SearchQuery", vocab.PropertyInbox, searchQueryService.ActivityPubInboxURL(searchQueryID), "/@search_:searchId/pub/inbox"},
		advertisedURL{"SearchQuery", vocab.PropertyOutbox, searchQueryService.ActivityPubOutboxURL(searchQueryID), "/@search_:searchId/pub/outbox"},
		advertisedURL{"SearchQuery", vocab.PropertyFollowers, searchQueryService.ActivityPubFollowersURL(searchQueryID), "/@search_:searchId/pub/followers"},
		advertisedURL{"SearchQuery", vocab.PropertyFollowing, searchQueryService.ActivityPubFollowingURL(searchQueryID), "/@search_:searchId/pub/following"},
	)

	// The @application actor has no per-collection accessors; Domain.GetJSONLD appends these
	// suffixes to the ActorID inline, so they are repeated here in the same order.
	domainService := service.NewDomain()
	actorID := domainService.ActorID()

	result = append(result,
		advertisedURL{"Domain", vocab.PropertyInbox, actorID + "/inbox", "/@application/inbox"},
		advertisedURL{"Domain", vocab.PropertyOutbox, actorID + "/outbox", "/@application/outbox"},
		advertisedURL{"Domain", vocab.PropertyFollowers, actorID + "/followers", "/@application/followers"},
		advertisedURL{"Domain", vocab.PropertyFollowing, actorID + "/following", "/@application/following"},
		advertisedURL{"Domain", vocab.PropertyLiked, actorID + "/liked", "/@application/liked"},
	)

	// Users need no database at all, so this actor walks its real document
	user := model.NewUser()
	user.UserID, err = primitive.ObjectIDFromHex(testUserID)
	require.Nil(t, err)
	user.ProfileURL = "https://example.com/@" + testUserID

	userDocument := user.GetJSONLD()

	userCollections := []struct {
		property string
		pattern  string
	}{
		{vocab.PropertyInbox, "/@:userId/pub/inbox"},
		{vocab.PropertyOutbox, "/@:userId/pub/outbox"},
		{vocab.PropertyFollowers, "/@:userId/pub/followers"},
		{vocab.PropertyFollowing, "/@:userId/pub/following"},
		{vocab.PropertyLiked, "/@:userId/pub/liked"},
	}

	for _, collection := range userCollections {
		result = append(result, advertisedURL{"User", collection.property, userDocument.GetString(collection.property), collection.pattern})
	}

	// RULE: An empty URL would resolve to nothing and quietly test nothing
	for _, advertised := range result {
		require.NotEmpty(t, advertised.url, "%s.%s produced an empty URL", advertised.actor, advertised.property)
	}

	return result
}

// makeTestRoutes builds the production route table on a bare Echo instance.
func makeTestRoutes() *echo.Echo {

	e := echo.New()

	// Registration never touches the Factory -- every handler closes over it and dereferences it
	// only once a request arrives -- so a nil Factory yields the real route table for free.
	makeApplicationRoutes(nil, e)

	return e
}

/******************************************
 * Route Table Helpers
 ******************************************/

// matchedGetRoute returns the pattern of the GET route that answers the provided path, or "" if no
// GET route matches it.
func matchedGetRoute(e *echo.Echo, path string) string {

	// Resolve through the real router, so this assertion cannot disagree with production routing
	request := httptest.NewRequest(http.MethodGet, path, nil)
	ctx := e.NewContext(request, httptest.NewRecorder())
	e.Router().Find(http.MethodGet, path, ctx)

	// Identify the matched handler by pointer instead of calling it.  These handlers close over a
	// nil Factory and would panic on execution; Echo substitutes one of its own fallbacks whenever
	// the path is unrouted (404) or registered only under a different verb (405).
	matched := reflect.ValueOf(ctx.Handler()).Pointer()

	if matched == reflect.ValueOf(echo.NotFoundHandler).Pointer() {
		return ""
	}

	if matched == reflect.ValueOf(echo.MethodNotAllowedHandler).Pointer() {
		return ""
	}

	return ctx.Path()
}

// methodsByPathSuffix indexes the registered routes beginning with the provided prefix, mapping the
// remainder of each path to its sorted HTTP methods.
func methodsByPathSuffix(e *echo.Echo, prefix string) map[string][]string {

	result := make(map[string][]string)

	for _, route := range e.Routes() {

		if !strings.HasPrefix(route.Path, prefix) {
			continue
		}

		suffix := strings.TrimPrefix(route.Path, prefix)
		result[suffix] = append(result[suffix], route.Method)
	}

	// Sort so that two route blocks written in a different order still compare equal
	for suffix := range result {
		sort.Strings(result[suffix])
	}

	return result
}

// urlPath reduces an advertised actor URL to the path that the router sees.
func urlPath(t *testing.T, value string) string {

	t.Helper()

	parsed, err := url.Parse(value)
	require.Nil(t, err)

	return parsed.Path
}
