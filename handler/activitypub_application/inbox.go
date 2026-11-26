package activitypub_application

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PostInbox receives an ActivityPub activity sent to the @application actor, verifies it, and routes it
func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.activitypub_application.PostInbox"

	activityService := factory.ActivityStream()

	// Verify the request through the shared inbox funnel, checked against admin-tier rules
	activity, err := activitypub.ReceiveRequest(ctx.Request(), activityService.AppClient(), activityService, factory.Rule(), session, primitive.NilObjectID)

	if err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// A request that passed validation proves that its host is running again
	factory.ReviveHost(activity.ActorID())

	// Route the activity to its handler
	context := Context{
		factory: factory,
		session: session,
	}

	if err := inboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Handling ActivityPub request")
	}

	return ctx.String(http.StatusOK, "")
}
