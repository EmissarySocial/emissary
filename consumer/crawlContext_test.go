package consumer

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// classifyOutcome returns just the outcome from classifyContext, for the tests that assert
// on classification alone
func classifyOutcome(t *testing.T, context streams.Document, err error) contextOutcome {

	t.Helper()

	outcome, retryAfter := classifyContext(context, err)

	// RULE: Only a rate limit carries a wait.  Any other outcome reporting one would
	// reschedule a task that no host ever asked to defer.
	if outcome != contextOutcomeRateLimited {
		require.Equal(t, time.Duration(0), retryAfter)
	}

	return outcome
}

// TestClassifyContext_Collections verifies that every ActivityStreams collection type is
// worth backfilling, so the crawl is never turned away by the spelling of a valid context
func TestClassifyContext_Collections(t *testing.T) {

	collectionTypes := []string{
		vocab.CoreTypeCollection,
		vocab.CoreTypeCollectionPage,
		vocab.CoreTypeOrderedCollection,
		vocab.CoreTypeOrderedCollectionPage,
	}

	for _, documentType := range collectionTypes {
		document := streams.NewDocument(mapof.Any{vocab.PropertyType: documentType})
		require.Equal(t, contextOutcomeBackfill, classifyOutcome(t, document, nil), "type: "+documentType)
	}
}

// TestClassifyContext_SuccessIsNeverAnError verifies that a document which loads cleanly and
// simply is not a collection is classified as Skip, rather than as a failure
func TestClassifyContext_SuccessIsNeverAnError(t *testing.T) {

	// BUG-150 Defect A: the original code read IsCollection before err and fell past it into
	// derp.Report, filing 301 contentless records in seven days.  A context that is not a
	// collection is a normal outcome on the open fediverse.
	nonCollectionTypes := []string{
		vocab.ObjectTypeNote,
		vocab.ObjectTypeArticle,
		vocab.ActorTypePerson,
		vocab.Unknown,
	}

	for _, documentType := range nonCollectionTypes {
		document := streams.NewDocument(mapof.Any{vocab.PropertyType: documentType})
		require.Equal(t, contextOutcomeSkip, classifyOutcome(t, document, nil), "type: "+documentType)
	}

	// A document with no "type" property at all, and an empty document, are both skipped
	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, streams.NewDocument(mapof.Any{}), nil))
	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, streams.NilDocument(), nil))
}

// TestClassifyContext_RemoteFailures verifies that every way a remote context can fail is
// classified as Skip, so that none of them reaches the error log
func TestClassifyContext_RemoteFailures(t *testing.T) {

	// BUG-150 Defect B: each of these is the remote's own failure and repeats identically on
	// every future crawl, so the InReplyTo crawl is this task's real answer to all of them.

	// The production case: 1,997 records in seven days, all of them a server answering
	// an ActivityPub request with 200 and an HTML page.
	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, streams.NilDocument(), htmlInsteadOfActivityPub("https://xxx.azyobuzi.net/contexts/1")))

	// RULE: This is why the classifier gates on err != nil and NOT on derp.IsClientError.
	// The HTML failure carries 500, so a client-error gate would have silenced none of it.
	require.Equal(t, http.StatusInternalServerError, derp.ErrorCode(htmlInsteadOfActivityPub("https://xxx.azyobuzi.net/contexts/1")))

	// Every other class of failure is skipped too
	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, streams.NilDocument(), derp.NotFound("test", "No such context")))
	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, streams.NilDocument(), derp.Forbidden("test", "Not for you")))
	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, streams.NilDocument(), derp.BadRequest("test", "Malformed context ID")))
	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, streams.NilDocument(), derp.Internal("test", "Remote server exploded")))

	// A dead domain answers no HTTP status at all -- DNS failure, refused connection, or timeout
	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, streams.NilDocument(), errors.New("dial tcp: no such host")))
}

// TestClassifyContext_RateLimited verifies the one failure that is still worth retrying: a 429
// throttles the HOST, so the whole task waits and tries again
func TestClassifyContext_RateLimited(t *testing.T) {

	// The host's own Retry-After is what reschedules the task
	outcome, retryAfter := classifyContext(streams.NilDocument(), tooManyRequests("90"))
	require.Equal(t, contextOutcomeRateLimited, outcome)
	require.Equal(t, 90*time.Second, retryAfter.Truncate(time.Second))

	// An HTTP-date is honored the same as delay-seconds.  The 121 is deliberate: RFC1123 has
	// only second granularity, so a 120 would truncate down to 119 and flake.
	outcome, retryAfter = classifyContext(streams.NilDocument(), tooManyRequests(time.Now().Add(121*time.Second).Format(time.RFC1123)))
	require.Equal(t, contextOutcomeRateLimited, outcome)
	require.Equal(t, 120*time.Second, retryAfter.Truncate(time.Second))

	// A 429 with no Retry-After still requeues, on derp's own one-hour default
	outcome, retryAfter = classifyContext(streams.NilDocument(), tooManyRequests(""))
	require.Equal(t, contextOutcomeRateLimited, outcome)
	require.Equal(t, time.Hour, retryAfter)

	// RULE: The duration survives the derp.Wrap that the client stack applies on the way up.
	// An unwrapped-only read would turn every rate limit into a permanent skip, and an
	// unwrapped-only duration would throttle every host to the one-hour default.
	wrapped := derp.Wrap(tooManyRequests("30"), "test", "Loading context collection", "https://example.social/ctx/1")
	outcome, retryAfter = classifyContext(streams.NilDocument(), wrapped)
	require.Equal(t, contextOutcomeRateLimited, outcome)
	require.Equal(t, 30*time.Second, retryAfter.Truncate(time.Second))

	// RULE: A rate limit never reschedules on a zero delay, whatever the host sends.  derp
	// clamps a negative, unparseable, or already-expired Retry-After to zero, and then
	// substitutes an hour -- so queue.Requeue can not spin against a throttling host.
	hostileHeaders := []string{
		"-30",
		"not-a-number",
		time.Now().Add(-time.Hour).Format(time.RFC3339),
	}

	for _, header := range hostileHeaders {
		outcome, retryAfter = classifyContext(streams.NilDocument(), tooManyRequests(header))
		require.Equal(t, contextOutcomeRateLimited, outcome, "header: "+header)
		require.Greater(t, retryAfter, time.Duration(0), "header: "+header)
	}
}

// TestClassifyContext_ErrorBeatsDocument verifies that an error arriving alongside a readable
// collection settles the outcome, so the document is never consulted first
func TestClassifyContext_ErrorBeatsDocument(t *testing.T) {

	// This is the ordering the original code had backwards (BUG-150).  A failed Load still
	// returns a readable Document, so asking it first answers from a value nobody sent.
	collection := streams.NewDocument(mapof.Any{vocab.PropertyType: vocab.CoreTypeOrderedCollection})

	require.Equal(t, contextOutcomeSkip, classifyOutcome(t, collection, derp.NotFound("test", "Gone")))

	// A 429 still wins over a readable collection, and still carries the host's wait
	outcome, retryAfter := classifyContext(collection, tooManyRequests("60"))
	require.Equal(t, contextOutcomeRateLimited, outcome)
	require.Equal(t, 60*time.Second, retryAfter.Truncate(time.Second))
}

// TestDerpWrapNil_IsAPhantomError pins the dependency behavior that makes classifyContext test
// err != nil rather than wrapping unconditionally
func TestDerpWrapNil_IsAPhantomError(t *testing.T) {

	// BUG-150 Defect A was built on this: a derp.Report reached on a success path files a
	// record with nothing in it to identify.
	phantom := derp.Wrap(nil, "consumer.CrawlContext", "Loading context collection")

	// Wrapping nil yields a NON-nil error, and derp.IsNil does not catch it either
	require.Error(t, phantom)
	require.False(t, derp.IsNil(phantom))

	// It carries status code 0, which is how these records are recognized in the error log
	require.Equal(t, 0, derp.ErrorCode(phantom))
}
