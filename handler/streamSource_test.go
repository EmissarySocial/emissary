package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

/******************************************
 * In-Memory Fakes
 ******************************************/

// webhookCollection records the criteria it was queried with, and returns no records
type webhookCollection struct {
	queried []exp.Predicate
}

// Iterator implements the data.Collection interface, capturing the criteria and walking nothing
func (c *webhookCollection) Iterator(criteria exp.Expression, _ ...option.Option) (data.Iterator, error) {

	// RULE: Report a match for every predicate.  AndExpression.Match stops at the first FALSE, so
	// a visitor that answered FALSE would capture one predicate and silently miss the rest.
	criteria.Match(func(predicate exp.Predicate) bool {
		c.queried = append(c.queried, predicate)
		return true
	})

	return &webhookIterator{}, nil
}

// Context implements the data.Collection interface
func (c *webhookCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *webhookCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *webhookCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Load implements the data.Collection interface. Unused by these tests.
func (c *webhookCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *webhookCollection) Save(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// Delete implements the data.Collection interface. Unused by these tests.
func (c *webhookCollection) Delete(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *webhookCollection) HardDelete(exp.Expression) error {
	return derp.Internal("test", "unused")
}

// webhookIterator walks no records. Implements data.Iterator.
type webhookIterator struct{}

// Next implements the data.Iterator interface, reporting an empty result
func (i *webhookIterator) Next(any) bool { return false }

// Count implements the data.Iterator interface
func (i *webhookIterator) Count() int { return 0 }

// Close implements the data.Iterator interface. It holds nothing.
func (i *webhookIterator) Close() error { return nil }

// Error implements the data.Iterator interface
func (i *webhookIterator) Error() error { return nil }

// webhookSession hands out a single shared webhookCollection
type webhookSession struct {
	collection *webhookCollection
}

// Collection implements the data.Session interface
func (s webhookSession) Collection(string) data.Collection { return s.collection }

// Context implements the data.Session interface
func (s webhookSession) Context() context.Context { return context.Background() }

// Close implements the data.Session interface. The stub holds no resources to release.
func (s webhookSession) Close() {}

// countingReader reports how many bytes were actually read out of a request body
type countingReader struct {
	remaining int
	read      int
}

// Read implements io.Reader, serving an endless stream of one byte at a time
func (r *countingReader) Read(target []byte) (int, error) {

	if (r.remaining <= 0) || (len(target) == 0) {
		return 0, io.EOF
	}

	target[0] = 'x'
	r.remaining--
	r.read++

	return 1, nil
}

/******************************************
 * Test Helpers
 ******************************************/

// postWebhook routes a request through the real echo route and returns what the handler answered
func postWebhook(t *testing.T, token string, body io.Reader) (*httptest.ResponseRecorder, *webhookCollection) {

	t.Helper()

	collection := &webhookCollection{}

	// A zero Factory is enough: the lookup reaches only session.Collection("StreamSource"), and a
	// record that matched would publish into a nil queue, which postcommit treats as a no-op
	factory := &service.Factory{}

	request := httptest.NewRequest(http.MethodPost, "/.streamsource/webhook/"+token, body)
	recorder := httptest.NewRecorder()

	// Routed through echo, rather than built by hand, so that the path in server.go and the
	// parameter this handler reads are proven to be the same name
	router := echo.New()
	router.POST("/.streamsource/webhook/:token", func(ctx echo.Context) error {
		return PostStreamSourceWebhook(
			&steranko.Context{Context: ctx},
			factory,
			webhookSession{collection: collection},
		)
	})

	router.ServeHTTP(recorder, request)

	return recorder, collection
}

/******************************************
 * One Answer For Every Outcome
 ******************************************/

// TestStreamSourceWebhook_AnswersIdentically pins the one answer this endpoint gives.  The token in
// the URL is the whole of the authorization and an admin may choose it by hand, so a response that
// varied by outcome would confirm a guess and then count the pages behind it.
func TestStreamSourceWebhook_AnswersIdentically(t *testing.T) {

	outcomes := map[string]string{
		"token too short":   "short",
		"unknown token":     "0123456789abcdef0123456789abcdef",
		"at the minimum":    strings.Repeat("a", 16),
		"wrong alphabet":    "................................",
		"absurdly long":     strings.Repeat("a", 4096),
		"looks like a path": "..%2f..%2fetc%2fpasswd",
	}

	for name, token := range outcomes {
		t.Run(name, func(t *testing.T) {

			recorder, _ := postWebhook(t, token, strings.NewReader(`{"ref":"refs/heads/main"}`))

			require.Equal(t, http.StatusAccepted, recorder.Code)
			require.Empty(t, recorder.Body.String(), "the response says nothing about what was found")
		})
	}
}

// TestStreamSourceWebhook_RefusedTokenNeverQueries confirms that a token too short to be real is
// turned away BEFORE the database is asked.  An empty token would otherwise match every record
// whose webhookToken was never set, which starts work rather than merely allowing it.
func TestStreamSourceWebhook_RefusedTokenNeverQueries(t *testing.T) {

	for _, token := range []string{"short", strings.Repeat("a", 15)} {

		recorder, collection := postWebhook(t, token, nil)

		require.Equal(t, http.StatusAccepted, recorder.Code, "token: %q", token)
		require.Empty(t, collection.queried, "a refused token reaches no query: %q", token)
	}
}

// TestStreamSourceWebhook_EmptyTokenNeverReachesTheHandler records the one request that does NOT
// answer 202: an empty token leaves no path segment to match, so echo answers 404 before any
// Emissary code runs.  That is stronger than the service's own refusal and reveals nothing -- the
// same 404 answers every unrouteable URL on the server -- but it IS a different answer, so it is
// pinned here rather than left to look like an exception somebody forgot.
func TestStreamSourceWebhook_EmptyTokenNeverReachesTheHandler(t *testing.T) {

	recorder, collection := postWebhook(t, "", nil)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Empty(t, collection.queried, "no token means no lookup")
}

/******************************************
 * The Token In The URL
 ******************************************/

// TestStreamSourceWebhook_LooksUpTheURLToken proves that the token in the path is the value queried
// for.  A parameter name that drifted from the route in server.go would read as empty, match
// nothing, and still answer 202 -- so every webhook on the server would silently stop working.
func TestStreamSourceWebhook_LooksUpTheURLToken(t *testing.T) {

	const token = "0123456789abcdef0123456789abcdef"

	recorder, collection := postWebhook(t, token, nil)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.NotEmpty(t, collection.queried, "the lookup must actually run")

	values := make(map[string]any, len(collection.queried))

	for _, predicate := range collection.queried {
		values[predicate.Field] = predicate.Value
	}

	require.Equal(t, token, values["config.webhookToken"])
	require.Equal(t, 0, values["deleteDate"], "a soft-deleted record must never be synchronized")
}

/******************************************
 * The Body
 ******************************************/

// TestStreamSourceWebhook_BoundsTheBody confirms that no more than the cap is ever read.  This
// route is public and unauthenticated, so an unbounded read is a memory-exhaustion target that
// costs an attacker exactly one request.
func TestStreamSourceWebhook_BoundsTheBody(t *testing.T) {

	body := &countingReader{remaining: streamSourceWebhookMaxBody * 4}

	recorder, _ := postWebhook(t, "0123456789abcdef0123456789abcdef", body)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Equal(t, streamSourceWebhookMaxBody, body.read, "the body is read up to the cap, and no further")
}

// TestStreamSourceWebhook_IgnoresTheBody confirms that the payload decides nothing.  Every forge
// signs and shapes it differently, and the origin is asked for the version anyway, so a handler
// that read it would buy nothing and lose the ability to be called from a post-receive hook.
func TestStreamSourceWebhook_IgnoresTheBody(t *testing.T) {

	const token = "0123456789abcdef0123456789abcdef"

	bodies := []string{
		"",
		"not json at all",
		`{"repository":{"full_name":"someone/else"}}`,
		`{"config":{"webhookToken":"a-different-token-entirely"}}`,
	}

	for _, body := range bodies {

		recorder, collection := postWebhook(t, token, strings.NewReader(body))

		require.Equal(t, http.StatusAccepted, recorder.Code, "body: %q", body)

		for _, predicate := range collection.queried {
			if predicate.Field == "config.webhookToken" {
				require.Equal(t, token, predicate.Value, "only the URL decides what is synchronized")
			}
		}
	}
}

// TestStreamSourceWebhookMaxBody confirms that the body cap matches the other unauthenticated
// webhooks on this server.  All three routes are reachable by anyone who guesses a URL, so a
// larger cap here would make this the cheapest memory-exhaustion target of the three.
func TestStreamSourceWebhookMaxBody(t *testing.T) {
	require.Equal(t, mailchimpWebhookMaxBody, streamSourceWebhookMaxBody)
}
