package camper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/benpate/digit"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// newWebFingerServer starts a loopback "home server" whose WebFinger endpoint
// publishes the provided links for every account, returning the server's URL.
func newWebFingerServer(t *testing.T, links []digit.Link) string {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// RULE: Only the WebFinger endpoint exists on this fake server
		if r.URL.Path != "/.well-known/webfinger" {
			http.NotFound(w, r)
			return
		}

		// Publish the configured links for whatever account was requested
		resource := digit.Resource{
			Subject: r.URL.Query().Get("resource"),
			Links:   links,
		}

		w.Header().Set("Content-Type", "application/jrd+json")
		require.NoError(t, json.NewEncoder(w).Encode(resource))
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server.URL
}

// TestGetTemplate_AllowPrivateIPs verifies that the SSRF guard blocks loopback lookups unless AllowPrivateIPs is set
func TestGetTemplate_AllowPrivateIPs(t *testing.T) {

	// Publish a FEP-3b86 Follow intent on a loopback home server
	followTemplate := "https://example.com/@me/intent/follow?object={object}&on-success={on-success}&on-cancel={on-cancel}"

	serverURL := newWebFingerServer(t, []digit.Link{{
		RelationType: IntentTypeFollow,
		Template:     followTemplate,
	}})

	account := serverURL + "/@alice"

	// RULE: By default, the SSRF guard refuses to dial loopback/private addresses,
	// so the lookup fails and no template is found
	blocked := New()
	require.Equal(t, "", blocked.GetTemplate("follow", account))

	// With AllowPrivateIPs, the WebFinger lookup succeeds and returns the template
	allowed := New(WithAllowPrivateIPs(true))
	require.Equal(t, followTemplate, allowed.GetTemplate("follow", account))

	// An explicit FALSE behaves exactly like the default
	explicit := New(WithAllowPrivateIPs(false))
	require.Equal(t, "", explicit.GetTemplate("follow", account))
}

// TestGetTemplate_LegacyRemoteFollow verifies that the OStatus "subscribe" link is rewritten into a Follow intent
func TestGetTemplate_LegacyRemoteFollow(t *testing.T) {

	// Publish only the legacy OStatus "subscribe" link (Mastodon-style remote follow)
	serverURL := newWebFingerServer(t, []digit.Link{{
		RelationType: digit.RelationTypeSubscribeRequest,
		Template:     "https://example.com/authorize_interaction?uri={uri}",
	}})

	account := serverURL + "/@alice"
	allowed := New(WithAllowPrivateIPs(true))

	// The legacy {uri} token is rewritten to {object} so PopulateTemplate can fill it
	require.Equal(t, "https://example.com/authorize_interaction?uri={object}", allowed.GetTemplate("follow", account))

	// RULE: The legacy fallback applies to Follow intents only
	require.Equal(t, "", allowed.GetTemplate("like", account))
}

// TestLookup verifies that Lookup returns the published WebFinger resource, and is refused by the SSRF guard by default
func TestLookup(t *testing.T) {

	followTemplate := "https://example.com/@me/intent/follow?object={object}"

	serverURL := newWebFingerServer(t, []digit.Link{{
		RelationType: IntentTypeFollow,
		Template:     followTemplate,
	}})

	account := serverURL + "/@alice"

	// Lookup with AllowPrivateIPs returns the published resource
	allowed := New(WithAllowPrivateIPs(true))
	resource, err := allowed.Lookup(account)
	require.NoError(t, err)
	require.Equal(t, followTemplate, resource.FindLink(IntentTypeFollow).Template)

	// RULE: Lookup without AllowPrivateIPs is refused by the SSRF guard
	blocked := New()
	_, err = blocked.Lookup(account)
	require.Error(t, err)
}

// TestGetTemplateFromKnownSoftware verifies the hard-coded intent templates for well-known server software
func TestGetTemplateFromKnownSoftware(t *testing.T) {

	camper := New()

	// The Emissary entry must match service.User.CreateIntentURL, which advertises
	// the same inbound route via WebFinger
	require.Equal(t,
		"/@me/intent/create?type={type}&name={name}&summary={summary}&content={content}&inReplyTo={inReplyTo}&on-success={on-success}&on-cancel={on-cancel}",
		camper.getTemplateFromKnownSoftware(vocab.ActivityTypeCreate, "Emissary"))

	// Mastodon-family servers share the /share path
	require.Equal(t, "/share?text={content}", camper.getTemplateFromKnownSoftware(vocab.ActivityTypeCreate, "mastodon"))
	require.Equal(t, "/share?text={content}", camper.getTemplateFromKnownSoftware(vocab.ActivityTypeCreate, "misskey"))

	// Mastodon-family servers open a remote object on their remote-interaction page
	require.Equal(t, "/authorize_interaction?uri={object}", camper.getTemplateFromKnownSoftware(intentNameObject, "mastodon"))
	require.Equal(t, "/authorize_interaction?uri={object}", camper.getTemplateFromKnownSoftware(intentNameObject, "Hometown"))
	require.Equal(t, "", camper.getTemplateFromKnownSoftware(intentNameObject, "misskey"))

	// RULE: Only Create and Object intents resolve through the known-software list
	require.Equal(t, "", camper.getTemplateFromKnownSoftware(vocab.ActivityTypeFollow, "emissary"))
	require.Equal(t, "", camper.getTemplateFromKnownSoftware(vocab.ActivityTypeLike, "mastodon"))
	require.False(t, isKnownSoftwareIntent(vocab.ActivityTypeLike))

	// Unknown software resolves to nothing
	require.Equal(t, "", camper.getTemplateFromKnownSoftware(vocab.ActivityTypeCreate, "friendster"))
}

// TestGetServername verifies that URLs, handles, and degenerate inputs all reduce to a server name
func TestGetServername(t *testing.T) {

	camper := New()

	// URLs reduce to their hostname
	require.Equal(t, "example.com", camper.getServername("https://example.com/@alice"))
	require.Equal(t, "example.com:8080", camper.getServername("http://example.com:8080/@alice"))

	// Handles reduce to everything after the last "@"
	require.Equal(t, "example.com", camper.getServername("@alice@example.com"))
	require.Equal(t, "example.com", camper.getServername("alice@example.com"))

	// Degenerate inputs fall through without panicking
	require.Equal(t, "", camper.getServername(""))
	require.Equal(t, "alice", camper.getServername("alice"))
}

// mastodonLinks are the three intent links mastodon.social published on 2026-10-23
var mastodonLinks = []digit.Link{
	{RelationType: digit.RelationTypeSubscribeRequest, Template: "https://mastodon.social/authorize_interaction?uri={uri}"},
	{RelationType: IntentTypeCreate, Template: "https://mastodon.social/share?text={content}"},
	{RelationType: IntentTypeObject, Template: "https://mastodon.social/authorize_interaction?uri={object}"},
}

// TestGetTemplate_ObjectFallback verifies that every intent but Create falls back to the Object intent
func TestGetTemplate_ObjectFallback(t *testing.T) {

	account := newWebFingerServer(t, mastodonLinks) + "/@alice"
	camper := New(WithAllowPrivateIPs(true))

	// Like, Announce, and Dislike are not published, so each opens the object instead
	for _, intent := range []string{"like", "announce", "dislike"} {
		require.Equal(t, "https://mastodon.social/authorize_interaction?uri={object}", camper.GetTemplate(intent, account), intent)
	}

	// RULE: The legacy "subscribe" link still wins for Follow, because it completes the follow
	require.Equal(t, "https://mastodon.social/authorize_interaction?uri={object}", camper.GetTemplate("follow", account))

	// Create uses its own template
	require.Equal(t, "https://mastodon.social/share?text={content}", camper.GetTemplate("create", account))
}

// TestGetTemplate_ObjectFallbackPrefersFollow verifies that the subscribe link is chosen over the Object link for Follow
func TestGetTemplate_ObjectFallbackPrefersFollow(t *testing.T) {

	account := newWebFingerServer(t, []digit.Link{
		{RelationType: IntentTypeObject, Template: "https://example.com/open?object={object}"},
		{RelationType: digit.RelationTypeSubscribeRequest, Template: "https://example.com/follow?uri={uri}"},
	}) + "/@alice"

	camper := New(WithAllowPrivateIPs(true))
	require.Equal(t, "https://example.com/follow?uri={object}", camper.GetTemplate("follow", account))
}

// TestGetTemplate_CreateNeverFallsBackToObject verifies that an Object link alone gives no Create template
func TestGetTemplate_CreateNeverFallsBackToObject(t *testing.T) {

	account := newWebFingerServer(t, []digit.Link{
		{RelationType: IntentTypeObject, Template: "https://example.com/open?object={object}"},
	}) + "/@alice"

	camper := New(WithAllowPrivateIPs(true))
	require.Equal(t, "", camper.GetTemplate("create", account))
	require.Equal(t, "https://example.com/open?object={object}", camper.GetTemplate("like", account))
}

// TestGetTemplate_AssumeSharePath verifies that the /share probe asks the account's server, and fills {content}
func TestGetTemplate_AssumeSharePath(t *testing.T) {

	// A server with no intents and no NodeInfo, but a working /share page
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/webfinger":
			w.Header().Set("Content-Type", "application/jrd+json")
			require.NoError(t, json.NewEncoder(w).Encode(digit.Resource{Subject: r.URL.Query().Get("resource")}))
		case "/share":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	camper := New(WithAllowPrivateIPs(true))
	require.Equal(t, server.URL+"/share?text={content}", camper.GetTemplate("create", server.URL+"/@alice"))
}

// requestCounter counts the requests a test server receives for each path
type requestCounter struct {
	lock  sync.Mutex
	paths map[string]int
}

// count returns the number of requests received for a path
func (counter *requestCounter) count(path string) int {
	counter.lock.Lock()
	defer counter.lock.Unlock()
	return counter.paths[path]
}

// add records one request for a path
func (counter *requestCounter) add(path string) {
	counter.lock.Lock()
	defer counter.lock.Unlock()
	counter.paths[path]++
}

// newSniffedServer starts a loopback home server whose WebFinger publishes the provided links,
// names its software through NodeInfo, and counts the requests made for each path
func newSniffedServer(t *testing.T, software string, links []digit.Link) (string, *requestCounter) {
	t.Helper()

	requests := &requestCounter{paths: map[string]int{}}
	var serverURL string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		requests.add(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {

		case "/.well-known/webfinger":
			require.NoError(t, json.NewEncoder(w).Encode(digit.Resource{Subject: r.URL.Query().Get("resource"), Links: links}))

		case "/.well-known/nodeinfo":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"links": []map[string]string{
				{"rel": "http://nodeinfo.diaspora.software/ns/schema/2.0", "href": serverURL + "/nodeinfo/2.0"},
			}}))

		case "/nodeinfo/2.0":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"software": map[string]string{"name": software}}))

		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	serverURL = server.URL

	return serverURL, requests
}

// TestGetTemplate_SniffedObjectFallback verifies that a server known only by its software still
// gets the Object intent for every intent but Create
func TestGetTemplate_SniffedObjectFallback(t *testing.T) {

	serverURL, _ := newSniffedServer(t, "mastodon", nil)
	account := serverURL + "/@alice"
	camper := New(WithAllowPrivateIPs(true))

	for _, intent := range []string{"like", "announce", "dislike", "follow", "undo"} {
		require.Equal(t, serverURL+"/authorize_interaction?uri={object}", camper.GetTemplate(intent, account), intent)
	}

	require.Equal(t, serverURL+"/share?text={content}", camper.GetTemplate("create", account))
}

// TestGetTemplate_SniffedUnknownSoftware verifies that software with no known templates gives nothing
func TestGetTemplate_SniffedUnknownSoftware(t *testing.T) {

	serverURL, _ := newSniffedServer(t, "friendster", nil)
	camper := New(WithAllowPrivateIPs(true))

	require.Equal(t, "", camper.GetTemplate("like", serverURL+"/@alice"))
}

// TestGetTemplate_RequestsEachDocumentOnce verifies that falling back to Object does not repeat
// the WebFinger or NodeInfo requests
func TestGetTemplate_RequestsEachDocumentOnce(t *testing.T) {

	serverURL, requests := newSniffedServer(t, "mastodon", nil)
	camper := New(WithAllowPrivateIPs(true))

	require.NotEqual(t, "", camper.GetTemplate("like", serverURL+"/@alice"))
	require.Equal(t, 1, requests.count("/.well-known/webfinger"))
	require.Equal(t, 1, requests.count("/.well-known/nodeinfo"))
	require.Equal(t, 1, requests.count("/nodeinfo/2.0"))
}

// TestGetTemplate_WebFingerObjectSkipsNodeInfo verifies that a published Object intent is used
// without asking NodeInfo
func TestGetTemplate_WebFingerObjectSkipsNodeInfo(t *testing.T) {

	serverURL, requests := newSniffedServer(t, "mastodon", mastodonLinks)
	camper := New(WithAllowPrivateIPs(true))

	require.Equal(t, "https://mastodon.social/authorize_interaction?uri={object}", camper.GetTemplate("like", serverURL+"/@alice"))
	require.Equal(t, 0, requests.count("/.well-known/nodeinfo"))
}

// TestGetTemplate_Reply verifies that Reply uses a Create that names the post being answered, or else Object
func TestGetTemplate_Reply(t *testing.T) {

	camper := New(WithAllowPrivateIPs(true))

	// A Create that takes {inReplyTo} answers the post directly
	emissary := newWebFingerServer(t, []digit.Link{
		{RelationType: IntentTypeCreate, Template: "https://example.com/@me/intent/create?content={content}&inReplyTo={inReplyTo}"},
		{RelationType: IntentTypeObject, Template: "https://example.com/open?object={object}"},
	}) + "/@alice"
	require.Equal(t, "https://example.com/@me/intent/create?content={content}&inReplyTo={inReplyTo}", camper.GetTemplate("reply", emissary))

	// A Create that cannot name the post gives way to Object, which opens the post to reply there
	mastodon := newWebFingerServer(t, mastodonLinks) + "/@alice"
	require.Equal(t, "https://mastodon.social/authorize_interaction?uri={object}", camper.GetTemplate("reply", mastodon))

	// With neither, there is nothing to offer
	nothing := newWebFingerServer(t, nil) + "/@alice"
	require.Equal(t, "", camper.GetTemplate("reply", nothing))
}

// TestPopulateTemplate verifies that every copy of a placeholder is filled, and spaces are encoded as %20
func TestPopulateTemplate(t *testing.T) {

	camper := New()
	data := url.Values{"content": {"hello world + more"}, "object": {"https://example.social/post/1"}}

	result := camper.PopulateTemplate("https://home.example/share?text={content}&again={content}&o={object}&missing={name}", data)

	require.Equal(t, "https://home.example/share?text=hello%20world%20%2B%20more&again=hello%20world%20%2B%20more&o=https%3A%2F%2Fexample.social%2Fpost%2F1&missing=", result)
}
