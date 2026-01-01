package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream) error {

	const location = "handler.activitypub_stream.PostInbox"

	// Verify the stream is an ActivityPub actor (based on the Template)
	actor := template.Actor

	if actor.IsNil() {
		return derp.NotFound(location, "Actor not found")
	}

	// Get an ActivityStream service for the Stream
	activityService := factory.ActivityStream(model.ActorTypeStream, stream.StreamID)

	// Create a new request context for the ActivityPub router
	context := Context{
		factory: factory,
		session: session,
		stream:  stream,
		actor:   &actor,
	}

	// Retrieve the activity from the request body
	if err := streamRouter.ReceiveAndHandle(context, ctx.Request(), activityService.Client()); err != nil {
		return derp.Wrap(err, location, "Error receiving ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
