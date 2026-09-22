package handler

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

/******************************************
 * StreamSource Webhook
 *
 * Public, unauthenticated, and reachable by anyone
 * who guesses the URL.  The token in the path is the
 * whole of the authorization, it is deliberately NOT
 * unique, and every outcome answers identically -- a
 * distinguishable response would say which tokens
 * exist and how many pages sit behind each one.
 * See GIT-MARKDOWN-TO-STREAM-CONTENT.md section 6.
 ******************************************/

// streamSourceWebhookMaxBody bounds an inbound webhook delivery, matching the cap on the
// Mailchimp and Stripe webhooks
const streamSourceWebhookMaxBody = 65535

// PostStreamSourceWebhook synchronizes every StreamSource record that carries the token in the URL
func PostStreamSourceWebhook(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	// RULE: Bound the body before discarding it.  This route is public and unauthenticated, so an
	// unbounded read is a memory-exhaustion target that costs an attacker one request.
	_, _ = io.Copy(io.Discard, io.LimitReader(ctx.Request().Body, streamSourceWebhookMaxBody))

	defer derp.ReportFunc(ctx.Request().Body.Close)

	// RULE: The payload is never read.  Every forge signs and shapes it differently, and the
	// origin is asked for the version anyway, so trusting it would buy nothing and cost the
	// ability to be called from a post-receive hook or a CI job.
	//
	// The error is deliberately discarded rather than reported: a token too short to look up is
	// what a prober sends, and filing each one would bury real defects in the error log.
	_ = factory.StreamSource().SyncByWebhookToken(session, ctx.Param("token"))

	// RULE: One response for every outcome -- a token too short to look up, an unknown token, a
	// known token matching nothing, and a known token matching forty records.  An admin may set
	// this token by hand, so it may well be something guessable, and a varying answer would
	// confirm the guess and then count the pages behind it.
	return ctx.NoContent(http.StatusAccepted)
}
