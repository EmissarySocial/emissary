package consumer

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// actorError itself is NOT unit-tested here: every branch but the rate-limited one now writes to
// the Following record, which needs a live *service.Factory (no consumer test builds one, and its
// SSE channel would block on a zero-valued service). That IS the fix -- a dead domain answers DNS
// or TLS errors, never 4xx, so the old "touches nothing" property is exactly what had to go. The
// policy it consults is classifyPollError below; the writes are covered in the service package.

// TestClassifyPollError pins the policy that decides what a failed poll means. Before BUG-148
// every one of these was queue.Error, so seven permanently-broken Following records were
// re-polled and re-reported 56 times each in 25 hours.
func TestClassifyPollError(t *testing.T) {

	// RULE: 429 means the HOST is throttling, so the task reschedules and the record is blameless
	require.Equal(t, pollOutcomeRateLimited, classifyPollError(tooManyRequests("45")))
	require.Equal(t, pollOutcomeRateLimited, classifyPollError(tooManyRequests("")))
	require.Equal(t, pollOutcomeRateLimited, classifyPollError(httpStatus(http.StatusTooManyRequests)))

	// RULE: 429 is a CLIENT error too, so ordering matters. If IsClientError were tested first,
	// every rate limit would be recorded against the record that got throttled.
	require.True(t, derp.IsClientError(httpStatus(http.StatusTooManyRequests)))

	// RULE: 410 is the server stating the account is deleted, so it needs no waiting period
	require.Equal(t, pollOutcomeGone, classifyPollError(httpStatus(http.StatusGone)))
	require.Equal(t, pollOutcomeGone, classifyPollError(derp.Gone("test", "deleted")))

	// RULE: 404 is NOT 410. A missing document is too easy to hit by misconfiguration, and
	// BUG-148's own Defect B was a 404 on a key that this server was wrongly withholding.
	require.Equal(t, pollOutcomeFailed, classifyPollError(httpStatus(http.StatusNotFound)))

	// RULE: 401/403 never shortcut to dead. Defect B's 401s were OUR defect, and abandoning
	// those follows would have made them unrecoverable once the key was fixed.
	require.Equal(t, pollOutcomeFailed, classifyPollError(httpStatus(http.StatusUnauthorized)))
	require.Equal(t, pollOutcomeFailed, classifyPollError(httpStatus(http.StatusForbidden)))

	// RULE: The dead-domain shapes must be RECORDED, not retried. These answer no HTTP status
	// at all, which is why classifying on 4xx alone left them touching nothing.
	require.Equal(t, pollOutcomeFailed, classifyPollError(errors.New("dial tcp: lookup gone.example: no such host")))
	require.Equal(t, pollOutcomeFailed, classifyPollError(errors.New("tls: handshake failure")))
	require.Equal(t, pollOutcomeFailed, classifyPollError(errors.New("context deadline exceeded")))
	require.Equal(t, pollOutcomeFailed, classifyPollError(derp.Internal("test", "Server exploded")))

	// Every status code lands somewhere, and only the two named codes get special treatment
	for code := range 600 {

		switch outcome := classifyPollError(httpStatus(code)); code {
		case http.StatusTooManyRequests:
			require.Equal(t, pollOutcomeRateLimited, outcome, "code %d", code)
		case http.StatusGone:
			require.Equal(t, pollOutcomeGone, outcome, "code %d", code)
		default:
			require.Equal(t, pollOutcomeFailed, outcome, "code %d", code)
		}
	}

	// A nil error is never passed here, but must not panic if it ever is
	require.NotPanics(t, func() { classifyPollError(nil) })
}

// TestFollowingStatusMessage checks the sentence that reaches the Following's owner. Until
// BUG-148 nothing told a user their follow had stopped working, so the message is the
// user-visible half of the fix and names the HTTP code for anyone reporting it.
func TestFollowingStatusMessage(t *testing.T) {

	// A refusal reads as a refusal, and names its code
	require.Contains(t, followingStatusMessage(derp.Unauthorized("test", "nope")), "401")
	require.Contains(t, followingStatusMessage(derp.Unauthorized("test", "nope")), "refused")
	require.Contains(t, followingStatusMessage(derp.Forbidden("test", "nope")), "403")
	require.Contains(t, followingStatusMessage(derp.Forbidden("test", "nope")), "refused")

	// A missing actor reads as missing
	require.Contains(t, followingStatusMessage(derp.NotFound("test", "gone")), "404")
	require.Contains(t, followingStatusMessage(derp.NotFound("test", "gone")), "could not be found")

	// Every other client error falls through to the generic sentence, still naming the code
	require.Contains(t, followingStatusMessage(derp.BadRequest("test", "malformed")), "400")
	require.Contains(t, followingStatusMessage(derp.BadRequest("test", "malformed")), "Unable to read")

	// RULE: The message is never empty, for ANY error -- an empty StatusMessage renders as a
	// red "failed" badge with no explanation at all.
	for code := 0; code <= 599; code++ {
		message := followingStatusMessage(httpStatus(code))
		require.NotEmpty(t, message, "code %d", code)
	}

	// A nil error is never passed here, but must not panic if it ever is
	require.NotPanics(t, func() { followingStatusMessage(nil) })
	require.NotEmpty(t, followingStatusMessage(nil))

	// The wrapping that every call site applies must not hide the code
	wrapped := derp.Wrap(derp.Unauthorized("test", "nope"), "test", "Loading ActivityPub Actor")
	require.Contains(t, followingStatusMessage(wrapped), "401")
}

// TestFollowingStatusMessage_Transport covers the failures that never reach a server at all.
// derp reports a DNS or TLS failure as 500, the same as a real server error, so naming the code
// would tell the owner "(500)" about a domain that does not resolve.
func TestFollowingStatusMessage_Transport(t *testing.T) {

	message := followingStatusMessage(errors.New("dial tcp: lookup gone.example: no such host"))
	require.Contains(t, message, "Could not reach this server")
	require.Contains(t, message, "no such host")
	require.NotContains(t, message, "500")

	// A deleted account says so plainly, rather than "may have been moved or deleted"
	require.Equal(t, "This account has been deleted (410).", followingStatusMessage(httpStatus(http.StatusGone)))

	// RULE: The message must fit model.Following's "statusMessage" schema, which rejects
	// anything over 1024 characters -- and a quoted root message has no bound of its own.
	huge := followingStatusMessage(errors.New(strings.Repeat("x", 100_000)))
	require.LessOrEqual(t, len(huge), statusMessageMaxLength)
	require.True(t, utf8.ValidString(huge))
}

// TestTruncate covers the bound on any text quoted into a status message.
func TestTruncate(t *testing.T) {

	// Short enough is returned unchanged
	require.Equal(t, "hello", truncate("hello", 10))
	require.Equal(t, "hello", truncate("hello", 5))
	require.Equal(t, "", truncate("", 10))

	// Too long is cut, marked, and never exceeds the limit
	require.Equal(t, "h...", truncate("hello", 4))

	// Below the width of the marker there is no room to mark anything, so it cuts hard
	require.Equal(t, "he", truncate("hello", 2))
	require.Equal(t, "", truncate("hello", 0))
	require.LessOrEqual(t, len(truncate(strings.Repeat("a", 5000), 100)), 100)

	// RULE: Cutting mid-rune would produce invalid UTF-8, which Mongo and the template both
	// carry straight through to the page.
	for _, value := range []string{"日本語のテキストです", "emoji 🙂🙂🙂🙂", "mixed ascii and 漢字"} {
		for maxLength := 1; maxLength <= len(value)+2; maxLength++ {
			result := truncate(value, maxLength)
			require.True(t, utf8.ValidString(result), "%q at %d produced invalid UTF-8", value, maxLength)
			require.LessOrEqual(t, len(result), maxLength, "%q at %d exceeded the limit", value, maxLength)
		}
	}

	// Degenerate limits do not panic or produce garbage. A negative length used to reach a
	// negative slice bound, which panics; reaching these assertions is the proof that it does not.
	require.Equal(t, "", truncate("hello", -1))
	require.Equal(t, "", truncate("hello", -1000))
	require.Equal(t, "", truncate("日本語", 1))
}

// TestAnsweredWithoutActor pins the one signal that separates "the server answered, and the BODY
// was the problem" from every other failure: a 2xx status on the response that actually arrived.
func TestAnsweredWithoutActor(t *testing.T) {

	// A 2xx answer that could not be read as an Actor, with the media type it arrived as
	status, contentType, answered := answeredWithoutActor(answeredWith(http.StatusOK, "text/html; charset=utf-8"))
	require.True(t, answered)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "text/html", contentType, "the charset parameter must not reach the owner's sentence")

	// Every 2xx counts, because any of them means the request completed
	for code := 200; code <= 299; code++ {
		_, _, answered := answeredWithoutActor(answeredWith(code, "text/html"))
		require.True(t, answered, "code %d", code)
	}

	// RULE: Nothing outside 2xx is this case. A 4xx or 5xx is a refusal or a server fault, and
	// a transport failure never produced a response at all.
	for _, code := range []int{0, 100, 301, 400, 404, 410, 429, 500, 503} {
		_, _, answered := answeredWithoutActor(answeredWith(code, "text/html"))
		require.False(t, answered, "code %d", code)
	}

	// An error carrying no HTTPError at all is never this case
	_, _, answered = answeredWithoutActor(errors.New("dial tcp: lookup gone.example: no such host"))
	require.False(t, answered)

	_, _, answered = answeredWithoutActor(derp.Internal("test", "Server exploded"))
	require.False(t, answered)

	// A missing Content-Type is reported as empty rather than guessed at
	_, contentType, answered = answeredWithoutActor(answeredWith(http.StatusOK, ""))
	require.True(t, answered)
	require.Empty(t, contentType)

	// RULE: The media type is written by the REMOTE server, so it is bounded. Unbounded, it
	// would push the explanation out of the owner's status message.
	_, contentType, answered = answeredWithoutActor(answeredWith(http.StatusOK, strings.Repeat("x", 100_000)))
	require.True(t, answered)
	require.LessOrEqual(t, len(contentType), contentTypeMaxLength)

	// A nil error must not panic
	require.NotPanics(t, func() { answeredWithoutActor(nil) })
}

// TestAnsweredWithoutActor_IsWhatRemoteProduces drives benpate/remote against a live server that
// answers HTML, so the rule above is pinned to what the library ACTUALLY returns. Without this the
// other tests would keep passing against a hand-built shape that remote had stopped producing.
func TestAnsweredWithoutActor_IsWhatRemoteProduces(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><h1>This domain is for sale</h1></body></html>`))
	}))

	defer server.Close()

	result := mapof.NewAny()
	transaction := remote.Get(server.URL).Accept("application/activity+json").Result(&result)
	transaction.AllowPrivateIPs(true)

	err := transaction.Send()
	require.NotNil(t, err, "a parked domain answering HTML must still be an error")

	// PRECONDITION: remote reports this as a 500 that is not a client error, which is why the
	// poller could not tell it apart from a dead server, and why the status alone is not enough.
	require.Equal(t, 500, derp.ErrorCode(err))
	require.False(t, derp.IsClientError(err))

	// The response that arrived is still in the chain, and it says 200
	status, contentType, answered := answeredWithoutActor(err)
	require.True(t, answered, "remote no longer carries the response status; the rule needs rewriting")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "text/html", contentType)
}

// TestFollowingStatusMessage_AnsweredWithoutActor covers the sentence for a server that answered
// perfectly well with something that is not an account. Saying it could not be reached is the
// opposite of what happened, and it is what the owner of all five BUG-151 follows was told.
func TestFollowingStatusMessage_AnsweredWithoutActor(t *testing.T) {

	message := followingStatusMessage(answeredWith(http.StatusOK, "text/html; charset=utf-8"))

	// RULE: Never claim the server was unreachable. It answered.
	require.NotContains(t, message, "Could not reach this server")
	require.NotContains(t, message, "200 OK", "quoting the status line as a reason is the defect")

	// It names what arrived, so the owner can tell a web page from a broken feed
	require.Contains(t, message, "200")
	require.Contains(t, message, "text/html")
	require.Contains(t, message, "did not return an account")

	// A feed body reaches the same sentence: while RSS is unsupported, it is equally "not an
	// account", and naming the media type is what distinguishes the two for anyone reporting it.
	feedMessage := followingStatusMessage(answeredWith(http.StatusOK, "application/rss+xml"))
	require.Contains(t, feedMessage, "application/rss+xml")
	require.NotContains(t, feedMessage, "Could not reach this server")

	// An unknown media type still produces a complete sentence
	bareMessage := followingStatusMessage(answeredWith(http.StatusOK, ""))
	require.NotEmpty(t, bareMessage)
	require.Contains(t, bareMessage, "200")

	// RULE: The bound still applies -- statusMessage is capped by model.Following's schema
	require.LessOrEqual(t, len(followingStatusMessage(answeredWith(http.StatusOK, strings.Repeat("x", 100_000)))), statusMessageMaxLength)

	// A genuine transport failure is untouched by this branch
	require.Contains(t, followingStatusMessage(errors.New("no such host")), "Could not reach this server")
}

// TestShouldReportPollError pins which failed polls reach the error log. A poll that fails for a
// reason this code already understands is recorded on the Following, and reporting it as well
// re-files the same fact on every cycle -- 1,891 records, 14.5% of the log, for five follows.
func TestShouldReportPollError(t *testing.T) {

	// RULE: A 2xx that carried no Actor is understood. The record says so; the log adds nothing.
	require.False(t, shouldReportPollError(answeredWith(http.StatusOK, "text/html")))
	require.False(t, shouldReportPollError(answeredWith(http.StatusNoContent, "")))

	// RULE: A 4xx is permanent and understood, and was already silent before this change
	require.False(t, shouldReportPollError(httpStatus(http.StatusNotFound)))
	require.False(t, shouldReportPollError(httpStatus(http.StatusUnauthorized)))
	require.False(t, shouldReportPollError(derp.Forbidden("test", "nope")))

	// RULE: Everything unexplained stays visible. A real 5xx, a DNS failure, a TLS failure and a
	// timeout are all things a human might need to act on.
	require.True(t, shouldReportPollError(httpStatus(http.StatusInternalServerError)))
	require.True(t, shouldReportPollError(httpStatus(http.StatusBadGateway)))
	require.True(t, shouldReportPollError(errors.New("dial tcp: lookup gone.example: no such host")))
	require.True(t, shouldReportPollError(errors.New("tls: handshake failure")))
	require.True(t, shouldReportPollError(derp.Internal("test", "Server exploded")))

	// A nil error is never passed here, but must not panic if it ever is
	require.NotPanics(t, func() { shouldReportPollError(nil) })
}

// httpStatus builds the derp.HTTPError shape that benpate/remote returns for one status code,
// which is exactly how a remote refusal reaches PollFollowing_Record in production
func httpStatus(code int) derp.HTTPError {
	return derp.HTTPError{
		Response: derp.HTTPResponseReport{StatusCode: code},
	}
}

// answeredWith builds the error that remote.Transaction.decodeResponseBody returns when a response
// arrives intact but cannot be decoded: an HTTPError carrying the real status, wrapped with
// WithInternalError, which is what makes the OUTER code 500 while the response underneath says 200.
func answeredWith(code int, contentType string) error {

	header := http.Header{}

	if contentType != "" {
		header.Set("Content-Type", contentType)
	}

	inner := derp.HTTPError{
		Response: derp.HTTPResponseReport{
			StatusCode: code,
			Status:     strconv.Itoa(code) + " " + http.StatusText(code),
			Header:     header,
		},
	}

	return derp.Wrap(inner, "remote.Transaction.decodeResponseBody", "HTML must be read into an io.Writer, *string, or *byte[]", derp.WithInternalError())
}
