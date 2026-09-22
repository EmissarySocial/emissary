package consumer

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/benpate/derp"
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

// httpStatus builds the derp.HTTPError shape that benpate/remote returns for one status code,
// which is exactly how a remote refusal reaches PollFollowing_Record in production
func httpStatus(code int) derp.HTTPError {
	return derp.HTTPError{
		Response: derp.HTTPResponseReport{StatusCode: code},
	}
}
