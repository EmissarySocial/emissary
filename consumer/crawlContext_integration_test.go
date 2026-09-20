package consumer

/******************************************
 * CrawlContext Integration Tests
 *
 * Every other test in this package hands classifyContext an
 * error built by hand.  These build one the way production
 * does: a real HTTP server, answered through the real client
 * stack, so the fixtures elsewhere are checked against the
 * shape a remote server actually produces (BUG-150).
 ******************************************/

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascacherules"
	"github.com/EmissarySocial/emissary/tools/asnormalizer"
	"github.com/EmissarySocial/emissary/tools/assanitizer"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/remote"
	"github.com/benpate/sherlock/activitypub"
	"github.com/benpate/sherlock/bridgyfed"
	"github.com/benpate/sherlock/tagspub"
	"github.com/benpate/sherlock/tombstone"
	"github.com/benpate/sherlock/webfinger"
	"github.com/stretchr/testify/require"
)

// documentClient builds the client stack that CrawlContext loads a context through
func documentClient() streams.Client {

	// RULE: This MUST mirror service.ActivityStream.Client, because the error shape these
	// tests pin is produced by the layers, not by any one of them.  The ascache layer is
	// the only omission, because it needs a live database and wraps nothing.

	// httptest serves from 127.0.0.1, which the transport's SSRF guard refuses by default
	const allowPrivateIPs = true

	client := activitypub.New(
		activitypub.WithUserAgent("emissary-test"),
		activitypub.WithAllowPrivateIPs(allowPrivateIPs),
	)

	client = tombstone.New(client)

	client = webfinger.New(client, remote.Option{
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.AllowPrivateIPs(allowPrivateIPs)
			return nil
		},
	})

	client = bridgyfed.New(client)
	client = tagspub.New(client)
	client = assanitizer.New(client, model.NamespaceEmissary)
	client = asnormalizer.New(client)

	return ascacherules.New(client)
}

// contextServer serves a single canned response at /contexts/1, and returns its URL
func contextServer(t *testing.T, status int, contentType string, body string, header http.Header) string {

	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		maps.Copy(w.Header(), header)
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)

		// The client closes the connection on any status it treats as an error, so a short
		// write here is expected and says nothing about the test.
		_, _ = fmt.Fprint(w, body)
	}))

	t.Cleanup(server.Close)

	return server.URL + "/contexts/1"
}

// errorChain renders a wrapped error as one line per layer, so a failure names the layer that broke
func errorChain(err error) []string {

	result := make([]string, 0)

	for err != nil {
		result = append(result, fmt.Sprintf("%T: %s", err, derp.Message(err)))
		err = errors.Unwrap(err)
	}

	return result
}

// TestIntegration_RateLimitedContext confirms that a host's own Retry-After survives every layer
// of the client stack and reaches the reschedule, instead of being replaced by derp's 1 hour default
func TestIntegration_RateLimitedContext(t *testing.T) {

	if testing.Short() {
		t.Skip("integration test: starts an HTTP server")
	}

	url := contextServer(t, http.StatusTooManyRequests, "application/activity+json",
		`{"error":"slow down"}`, http.Header{"Retry-After": []string{"30"}})

	context, err := documentClient().Load(url)

	t.Log("error chain: ", errorChain(err))

	outcome, retryAfter := classifyContext(context, err)

	require.Equal(t, contextOutcomeRateLimited, outcome)
	require.Equal(t, 30*time.Second, retryAfter, "the host asked for 30s; 1h means the duration was lost in the chain")
	require.Equal(t, http.StatusTooManyRequests, derp.ErrorCode(err))
}

// TestIntegration_HTMLInsteadOfActivityPub is BUG-150's largest error class, built by a real server:
// a 200 response carrying an HTML page, which derp reports as 500 rather than any 4xx
func TestIntegration_HTMLInsteadOfActivityPub(t *testing.T) {

	if testing.Short() {
		t.Skip("integration test: starts an HTTP server")
	}

	url := contextServer(t, http.StatusOK, "text/html; charset=utf-8",
		"<!DOCTYPE html><html><body>Hello</body></html>", nil)

	context, err := documentClient().Load(url)

	t.Log("error chain: ", errorChain(err))

	outcome, retryAfter := classifyContext(context, err)

	require.Equal(t, contextOutcomeSkip, outcome)
	require.Equal(t, time.Duration(0), retryAfter)

	// RULE: This is the assertion that makes derp.IsClientError the wrong gate for this bug.
	// The response really was a 200, and the error really is a 500, so no 4xx test matches it.
	require.Equal(t, http.StatusInternalServerError, derp.ErrorCode(err))
	require.False(t, derp.IsClientError(err))
}

// TestIntegration_NonCollectionContext is BUG-150's Defect A, built by a real server: a context
// that loads perfectly well and simply is not a collection, which is a success and not an error
func TestIntegration_NonCollectionContext(t *testing.T) {

	if testing.Short() {
		t.Skip("integration test: starts an HTTP server")
	}

	url := contextServer(t, http.StatusOK, "application/activity+json",
		`{"@context":"https://www.w3.org/ns/activitystreams","type":"Note","content":"Hello"}`, nil)

	context, err := documentClient().Load(url)

	require.NoError(t, err)
	require.False(t, context.IsCollection())

	outcome, retryAfter := classifyContext(context, err)

	require.Equal(t, contextOutcomeSkip, outcome)
	require.Equal(t, time.Duration(0), retryAfter)
}
