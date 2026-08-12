package camper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	// RULE: Only Create intents resolve through the known-software list
	require.Equal(t, "", camper.getTemplateFromKnownSoftware(vocab.ActivityTypeFollow, "emissary"))
	require.Equal(t, "", camper.getTemplateFromKnownSoftware(vocab.ActivityTypeLike, "mastodon"))

	// Unknown software resolves to nothing
	require.Equal(t, "", camper.getTemplateFromKnownSoftware(vocab.ActivityTypeCreate, "friendster"))
}

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
