package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PostInbox receives an inbound ActivityPub activity in a Stream actor's inbox, verifies it, and
// routes it to the matching handler.
func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.PostInbox"

	// Verify the stream is an ActivityPub actor (based on the Template)
	actor := template.Actor

	if actor.IsNil() {
		return derp.NotFound(location, "Actor not found")
	}

	// Get an ActivityStream service for the Stream
	activityService := factory.ActivityStream()
	client := activityService.StreamClient(stream.StreamID)

	// Create a new request context for the ActivityPub router
	context := Context{
		factory: factory,
		session: session,
		stream:  stream,
		actor:   &actor,
	}

	// Retrieve the activity through the canonical inbox receive funnel (Stage-1 validators + the
	// reserved-namespace sanitizer), evaluated against admin-tier rules (NilObjectID) -- Stage 1 of
	// the block gate (D5). The verifier keeps signature verification inside Emissary's client stack,
	// so it inherits the cache, the rules gate, and the private-IP policy (BUG-19).
	activity, err := activitypub.ReceiveRequest(ctx.Request(), client, activityService, factory.Rule(), session, primitive.NilObjectID)

	if err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// Route the activity to the appropriate handler
	if err := streamRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Handling ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
