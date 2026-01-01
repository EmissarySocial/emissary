package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

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
	activityService := factory.ActivityStream(model.ActorTypeUser, user.UserID)

	// Retrieve the activity from the request body
	if err := inboxRouter.ReceiveAndHandle(context, ctx.Request(), activityService.Client()); err != nil {
		return derp.Wrap(err, location, "Unable to handle ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
