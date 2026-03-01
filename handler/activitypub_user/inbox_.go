package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collection"
	"github.com/benpate/hannibal/router"
	"github.com/benpate/steranko"
)

func GetInboxCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	inboxService := factory.Inbox()

	return collection.Serve(ctx,
		inboxService.CollectionID(user.UserID),
		inboxService.CollectionCount(session, user.UserID),
		inboxService.CollectionIterator(session, user.UserID),
		collection.WithSSEEndpoint(user.ActivityPubSSEEndpointMLS()),
	)
}

func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.PostInbox"

	// Create a new Context
	context := Context{
		context: ctx,
		factory: factory,
		session: session,
		user:    user,
	}

	// Get ActivityStream service for this User
	client := factory.ActivityStream().UserClient(user.UserID)

	// Receive the activity from the request (with optional options)
	activity, err := router.ReceiveRequest(ctx.Request(), client)

	if err != nil {
		return derp.Wrap(err, location, "Unable to receive ActivityPub request")
	}

	//
	// ADDITIONAL VALIDATION LOGIC GOES HERE...
	//

	// Route the activity to the appropriate handlers (based on activityType and objectType)
	if err := inboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Unable to handle ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
