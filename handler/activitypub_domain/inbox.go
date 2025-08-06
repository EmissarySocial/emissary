package activitypub_domain

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/inbox"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.activitypub_domain.PostInbox"

	activityService := factory.ActivityStream(model.ActorTypeSearchDomain, primitive.NilObjectID)

	// Retrieve the activity from the request body
	activity, err := inbox.ReceiveRequest(ctx.Request(), activityService.Client())

	if err != nil {
		return derp.Wrap(err, location, "Error parsing ActivityPub request")
	}

	// Create a new request context for the ActivityPub router
	context := Context{
		factory: factory,
	}

	// Handle the ActivityPub request
	if err := inboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Error handling ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
