package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/inbox"
	"github.com/benpate/steranko"
)

func PostInbox(ctx *steranko.Context, factory *domain.Factory, session data.Session, user *model.User) error {

	const location = "handler.activitypub_user.PostInbox"

	// RULE: Only public users can be queried
	if !user.IsPublic {
		return derp.NotFoundError(location, "")
	}

	// Retrieve the activity from the request body
	activity, err := inbox.ReceiveRequest(ctx.Request(), factory.ActivityStream())

	if err != nil {
		return derp.Wrap(err, location, "Error parsing ActivityPub request")
	}

	// Create a new Context
	context := Context{
		factory: factory,
		user:    user,
	}

	// Handle the ActivityPub request
	if err := inboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Error handling ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
