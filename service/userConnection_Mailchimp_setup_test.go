package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/mailchimp"
	"github.com/benpate/remote"
	"github.com/stretchr/testify/require"
)

// testMailchimpCredential is shaped like a Mailchimp key, and is assembled from halves so
// that a secret scanner does not report the file as holding a live one
const testMailchimpCredential = "0123456789abcdef0123456789abcdef" + "-us6"

/******************************************
 * Mailchimp Audience Setup
 *
 * These exercise ORDERING and IDEMPOTENCY, not HTTP. The wire format is
 * covered by tools/mailchimp; what matters here is that setup can run twice,
 * that it stops at the right place when a call fails, and that a webhook it
 * cannot install does not take the connection down with it.
 ******************************************/

// mailchimpRecorder is an httptest server that answers Mailchimp's setup calls and
// remembers which ones were made
type mailchimpRecorder struct {
	server    *httptest.Server
	requests  []string
	fields    string
	webhooks  string
	bodies    map[string]string // the last request body sent to each "METHOD /path"
	failPaths map[string]int
}

// newMailchimpRecorder returns a recorder that reports an audience already carrying
// nothing, so that every create-if-missing call has to create
func newMailchimpRecorder() *mailchimpRecorder {

	recorder := &mailchimpRecorder{
		fields:    `{"merge_fields":[]}`,
		webhooks:  `{"webhooks":[]}`,
		bodies:    make(map[string]string),
		failPaths: make(map[string]int),
	}

	recorder.server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

		// The Client addresses Mailchimp's real API root, so the version prefix is trimmed
		// here rather than repeated in every expectation.
		key := request.Method + " " + strings.TrimPrefix(request.URL.Path, "/3.0")
		recorder.requests = append(recorder.requests, key)

		if body, err := io.ReadAll(request.Body); err == nil {
			recorder.bodies[key] = string(body)
		}

		if statusCode, ok := recorder.failPaths[key]; ok {
			response.WriteHeader(statusCode)
			return
		}

		response.Header().Set("Content-Type", "application/json")

		switch {

		case strings.HasSuffix(request.URL.Path, "/merge-fields"):
			if request.Method == http.MethodPost {
				_, _ = response.Write([]byte(`{"merge_id":9,"tag":"EMISSARYID"}`))
				return
			}
			_, _ = response.Write([]byte(recorder.fields))

		case strings.HasSuffix(request.URL.Path, "/webhooks"):
			if request.Method == http.MethodPost {
				_, _ = response.Write([]byte(`{"id":"wh1","url":"whatever"}`))
				return
			}
			_, _ = response.Write([]byte(recorder.webhooks))

		default:
			_, _ = response.Write([]byte(`{"lists":[]}`))
		}
	}))

	return recorder
}

// count returns how many times the provided "METHOD /path" was requested
func (recorder *mailchimpRecorder) count(key string) int {

	result := 0

	for _, request := range recorder.requests {
		if request == key {
			result++
		}
	}

	return result
}

// newSetupService returns a UserConnection service and a connection ready for setup,
// both pointed at the provided recorder
func newSetupService(t *testing.T, recorder *mailchimpRecorder) (UserConnection, mailchimp.Client, model.UserConnection) {

	t.Helper()

	service := UserConnection{
		encryptionKey: testDomainCipher,
		host:          "https://example.com",
	}

	userConnection := model.NewUserConnection()
	userConnection.Type = model.UserConnectionTypeMailchimp
	userConnection.IsActive.Set(true)
	userConnection.Vault.SetString(model.UserConnectionVaultWebhookSecret, "the-webhook-secret")

	client, err := mailchimp.New(testMailchimpCredential, "us6", redirectToTestServer(t, recorder.server.URL))
	require.NoError(t, err)

	return service, client, userConnection
}

// redirectToTestServer sends a real Client's requests to an httptest server, rewriting only
// the host so that every path, header, and body under test is the one Mailchimp would see
func redirectToTestServer(t *testing.T, serverURL string) remote.Option {

	t.Helper()

	parsed, err := url.Parse(serverURL)
	require.NoError(t, err)

	return remote.Option{

		// remote's SSRF guard blocks loopback by default, which is where httptest lives
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.AllowPrivateIPs(true)
			return nil
		},

		ModifyRequest: func(_ *remote.Transaction, request *http.Request) *http.Response {
			request.URL.Scheme = parsed.Scheme
			request.URL.Host = parsed.Host
			return nil
		},
	}
}

// testAudience returns the audience that Mailchimp confirmed a pasted ID names
func testAudience() mailchimp.Audience {
	return mailchimp.Audience{ID: "abc123", Name: "Newsletter"}
}

// TestMailchimpSetup_InstallsEverythingAndMarksReady walks the happy path
func TestMailchimpSetup_InstallsEverythingAndMarksReady(t *testing.T) {

	recorder := newMailchimpRecorder()
	defer recorder.server.Close()

	service, client, userConnection := newSetupService(t, recorder)

	require.NoError(t, service.mailchimp_setup(client, &userConnection, testAudience()))

	require.Equal(t, model.UserConnectionStatusReady, userConnection.Status)
	require.Equal(t, "abc123", userConnection.Data.GetString(model.UserConnectionDataAudienceID))
	require.Equal(t, "Newsletter", userConnection.Data.GetString(model.UserConnectionDataAudienceName), "the name is what makes a pasted ID checkable")
	require.Equal(t, "wh1", userConnection.Data.GetString(model.UserConnectionDataWebhookID))

	require.Equal(t, 1, recorder.count("POST /lists/abc123/merge-fields"))

	// D42: Emissary applies no tag.  EMISSARYID already marks exactly the members it pushed,
	// and a tag call would cost a second request per follower on 1.2's hot path.
	require.Equal(t, 0, recorder.count("GET /lists/abc123/segments"), "no tag is created")
	require.Equal(t, 0, recorder.count("POST /lists/abc123/segments"), "no tag is created")
	require.Equal(t, 1, recorder.count("POST /lists/abc123/webhooks"))
}

// TestMailchimpSetup_NeverRegistersTheApiSource guards the echo loop at the call site that
// decides it
func TestMailchimpSetup_NeverRegistersTheApiSource(t *testing.T) {

	// tools/mailchimp pins what CreateWebhook SENDS; this pins what this service ASKS FOR.
	// Only the second one can catch a `sources` value edited here, and registering `api`
	// would make every member Emissary adds arrive straight back at its own handler (D12).

	recorder := newMailchimpRecorder()
	defer recorder.server.Close()

	service, client, userConnection := newSetupService(t, recorder)

	require.NoError(t, service.mailchimp_setup(client, &userConnection, testAudience()))

	var sent struct {
		Events  mailchimp.WebhookEvents  `json:"events"`
		Sources mailchimp.WebhookSources `json:"sources"`
	}

	require.NoError(t, json.Unmarshal([]byte(recorder.bodies["POST /lists/abc123/webhooks"]), &sent))

	require.True(t, sent.Sources.User)
	require.True(t, sent.Sources.Admin)
	require.False(t, sent.Sources.API, "registering `api` echoes every write back at us (D12)")

	require.True(t, sent.Events.Subscribe)
	require.True(t, sent.Events.Unsubscribe)
	require.True(t, sent.Events.UpEmail)
	require.False(t, sent.Events.Cleaned, "bounces are a separate project")
	require.False(t, sent.Events.Profile)
	require.False(t, sent.Events.Campaign)
}

// TestMailchimpSetup_IsIdempotent is the guard against a second webhook
func TestMailchimpSetup_IsIdempotent(t *testing.T) {

	// Setup runs again on every reconnect, and after any partial failure.  A second webhook
	// on the same URL makes Mailchimp deliver every event twice, which is two passes through
	// Follower.Save per subscribe -- forever, with nothing reporting it.

	recorder := newMailchimpRecorder()
	defer recorder.server.Close()

	service, client, userConnection := newSetupService(t, recorder)

	require.NoError(t, service.mailchimp_setup(client, &userConnection, testAudience()))

	// Now Mailchimp reports everything the first run created
	callbackURL, err := service.mailchimp_callbackURL(&userConnection)
	require.NoError(t, err)

	recorder.fields = `{"merge_fields":[{"merge_id":9,"tag":"EMISSARYID"}]}`
	recorder.webhooks = `{"webhooks":[{"id":"wh1","url":"` + callbackURL + `"}]}`

	require.NoError(t, service.mailchimp_setup(client, &userConnection, testAudience()))

	require.Equal(t, 1, recorder.count("POST /lists/abc123/merge-fields"), "the EMISSARYID field must be created once")
	require.Equal(t, 1, recorder.count("POST /lists/abc123/webhooks"), "a second webhook doubles every delivery")

	require.Equal(t, "wh1", userConnection.Data.GetString(model.UserConnectionDataWebhookID))
}

// TestMailchimpSetup_WebhookFailureDoesNotBlockTheConnection pins D40
func TestMailchimpSetup_WebhookFailureDoesNotBlockTheConnection(t *testing.T) {

	// The webhook is the only step that asks Mailchimp to reach BACK at this server, which a
	// development machine cannot receive.  Gating READY on it would mean this feature could
	// only ever be built behind a tunnel.  §1.4 makes the gate strict; until then a failed
	// install is recorded as an empty webhookId and the connection still works outbound.

	recorder := newMailchimpRecorder()
	defer recorder.server.Close()

	recorder.failPaths["POST /lists/abc123/webhooks"] = http.StatusBadRequest

	service, client, userConnection := newSetupService(t, recorder)

	require.NoError(t, service.mailchimp_setup(client, &userConnection, testAudience()))

	require.Equal(t, model.UserConnectionStatusReady, userConnection.Status, "the connection is set up (D40)")
	require.Equal(t, "Newsletter", userConnection.Data.GetString(model.UserConnectionDataAudienceName))
	require.False(t, userConnection.HasWebhook(), "an install that failed leaves no webhookId")
}

// TestMailchimpSetup_MergeFieldFailureDoesBlock confirms D40's carve-out is narrow
func TestMailchimpSetup_MergeFieldFailureDoesBlock(t *testing.T) {

	// This is an ordinary outbound call that works the same everywhere, so there is no reason
	// to tolerate it failing -- and a connection without EMISSARYID cannot link a member back
	// to the Follower it came from.

	for _, path := range []string{"POST /lists/abc123/merge-fields"} {

		t.Run(path, func(t *testing.T) {

			recorder := newMailchimpRecorder()
			defer recorder.server.Close()

			recorder.failPaths[path] = http.StatusBadRequest

			service, client, userConnection := newSetupService(t, recorder)

			require.Error(t, service.mailchimp_setup(client, &userConnection, testAudience()))
			require.NotEqual(t, model.UserConnectionStatusReady, userConnection.Status)
			require.False(t, userConnection.IsReady())
		})
	}
}

// TestMailchimpCallbackURL_RefusesAnUnmintedSecret keeps an unauthenticated address out of
// a third party's hands
func TestMailchimpCallbackURL_RefusesAnUnmintedSecret(t *testing.T) {

	// Publishing a callback with no secret would hand Mailchimp an address that nothing can
	// authorize -- and every delivery it carried would be rejected by VerifyWebhookSecret.

	service := UserConnection{encryptionKey: testDomainCipher, host: "https://example.com"}

	userConnection := model.NewUserConnection()
	userConnection.Type = model.UserConnectionTypeMailchimp

	_, err := service.mailchimp_callbackURL(&userConnection)
	require.Error(t, err)
}

// TestMailchimpCallbackURL_CarriesTheSecret confirms the address Mailchimp is given
func TestMailchimpCallbackURL_CarriesTheSecret(t *testing.T) {

	service := UserConnection{encryptionKey: testDomainCipher, host: "https://example.com"}

	userConnection := model.NewUserConnection()
	userConnection.Type = model.UserConnectionTypeMailchimp
	userConnection.Vault.SetString(model.UserConnectionVaultWebhookSecret, "the-webhook-secret")

	result, err := service.mailchimp_callbackURL(&userConnection)

	require.NoError(t, err)
	require.Equal(t, "https://example.com/.mailchimp/webhook/"+userConnection.UserConnectionID.Hex()+"?secret=the-webhook-secret", result)
}

// TestMailchimpSetup_RefusesAnUndeliverableCallback is the local-development case, and it
// must stay non-fatal
func TestMailchimpSetup_RefusesAnUndeliverableCallback(t *testing.T) {

	// Nothing on the public internet can reach `localhost`, so asking Mailchimp to deliver
	// there is a round trip that can only fail. Refusing early says why; failing at Mailchimp
	// does not. The connection still finishes and still syncs outward (D40) -- which is the
	// whole reason this feature can be built on a laptop at all.

	recorder := newMailchimpRecorder()
	defer recorder.server.Close()

	service, client, userConnection := newSetupService(t, recorder)
	service.host = "http://localhost:8080"

	require.NoError(t, service.mailchimp_setup(client, &userConnection, testAudience()))

	require.Equal(t, model.UserConnectionStatusReady, userConnection.Status, "the connection is still set up")
	require.False(t, userConnection.HasWebhook())

	require.Equal(t, 0, recorder.count("GET /lists/abc123/webhooks"), "Mailchimp is never asked")
	require.Equal(t, 0, recorder.count("POST /lists/abc123/webhooks"))
}
