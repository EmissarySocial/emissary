package handler

import (
	"io"
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Mailchimp Webhook
 *
 * Public, unauthenticated, and reachable by anyone who guesses the URL. The
 * connection's own secret is the whole of the authorization, so every failure
 * answers identically -- a distinct response would turn this route into an
 * oracle for which connections exist. See MAILING-LISTS.md 1.3.
 ******************************************/

// mailchimpWebhookMaxBody bounds an inbound Mailchimp delivery, matching the cap on the
// Stripe webhook
const mailchimpWebhookMaxBody = 65535

// PostMailchimpWebhook applies an inbound Mailchimp event to the connection named in the URL
func PostMailchimpWebhook(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostMailchimpWebhook"

	// RULE: bound the body before reading it. This route is public and unauthenticated, so an
	// unbounded ReadAll is a memory-exhaustion target that costs an attacker one request.
	body, err := io.ReadAll(io.LimitReader(ctx.Request().Body, mailchimpWebhookMaxBody))

	if err != nil {
		return mailchimpWebhookResponse(ctx)
	}

	defer derp.ReportFunc(ctx.Request().Body.Close)

	// Load the connection named in the URL
	userConnection, err := mailchimpWebhookConnection(ctx, factory, session)

	if err != nil {
		return mailchimpWebhookResponse(ctx)
	}

	// RULE: the presented secret is the authorization. VerifyWebhookSecret also refuses a
	// connection that is switched off or flagged for reconnection, so a webhook that outlived
	// its connection cannot act.
	if !factory.UserConnection().VerifyWebhookSecret(&userConnection, ctx.QueryParam("secret")) {
		return mailchimpWebhookResponse(ctx)
	}

	// Past this point the caller has PROVEN they hold the secret, so a real failure can be
	// reported as one: it tells them nothing they did not already know, and returning it
	// rolls the transaction back instead of committing half of an event.

	// Mailchimp posts form-encoded, not JSON
	values, err := url.ParseQuery(string(body))

	if err != nil {
		return derp.Wrap(err, location, "Unable to parse webhook body", derp.WithBadRequest())
	}

	payload := make(mapof.String, len(values))

	for name := range values {
		payload[name] = values.Get(name)
	}

	// Apply the event
	if err := factory.UserConnection().Mailchimp_ReceiveWebhook(session, &userConnection, payload.GetString("type"), payload); err != nil {
		return derp.Wrap(err, location, "Applying Mailchimp webhook", userConnection.UserConnectionID)
	}

	return mailchimpWebhookResponse(ctx)
}

// mailchimpWebhookConnection loads the UserConnection named in the request URL
func mailchimpWebhookConnection(ctx *steranko.Context, factory *service.Factory, session data.Session) (model.UserConnection, error) {

	const location = "handler.mailchimpWebhookConnection"

	result := model.NewUserConnection()

	userConnectionID, err := primitive.ObjectIDFromHex(ctx.Param("userConnectionId"))

	if err != nil {
		return result, derp.BadRequest(location, "Invalid connection ID in URL")
	}

	// RULE: the lookup names the connection AND its service, so a Mailchimp delivery cannot
	// reach a connection to something else that happens to share an ID space.
	criteria := exp.
		Equal("_id", userConnectionID).
		AndEqual("type", model.UserConnectionTypeMailchimp)

	if err := factory.UserConnection().Load(session, criteria, &result); err != nil {
		return result, derp.Wrap(err, location, "Connection not found")
	}

	return result, nil
}

// mailchimpWebhookResponse answers every delivery identically, whatever happened
func mailchimpWebhookResponse(ctx *steranko.Context) error {

	// RULE: one response for every AUTHORIZATION outcome -- unknown connection, wrong secret,
	// paused connection, or success. ObjectIDs carry a timestamp and are partly predictable,
	// so a distinguishable answer would let someone enumerate which connections exist and
	// which are live. Failures after the secret is verified are reported normally.
	return ctx.NoContent(http.StatusOK)
}
