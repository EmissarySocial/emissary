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

func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.PostInbox"

	// Verify the stream is an ActivityPub actor (based on the Template)
	actor := template.Actor

	if actor.IsNil() {
		return derp.NotFound(location, "Actor not found")
	}

	// Get an ActivityStream service for the Stream
	client := factory.ActivityStream().StreamClient(stream.StreamID)

	// Create a new request context for the ActivityPub router
	context := Context{
		factory: factory,
		session: session,
		stream:  stream,
		actor:   &actor,
	}

	// Retrieve the activity from the request body, gated by the canonical inbox validator chain
	// evaluated against admin-tier rules (NilObjectID) -- Stage 1 of the block gate (D5).
	if err := streamRouter.ReceiveAndHandle(context, ctx.Request(), client, activitypub.InboxValidators(factory.Rule(), session, primitive.NilObjectID)); err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
