package activitypub_domain

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.activitypub_domain.PostInbox"

	client := factory.ActivityStream().SearchDomainClient()

	// Create a new request context for the ActivityPub router
	context := Context{
		factory: factory,
		session: session,
	}

	// Retrieve the activity from the request body, gated by the canonical inbox validator chain
	// evaluated against admin-tier rules (NilObjectID) -- Stage 1 of the block gate (D5).
	if err := inboxRouter.ReceiveAndHandle(context, ctx.Request(), client, activitypub.InboxValidators(factory.Rule(), session, primitive.NilObjectID)); err != nil {
		return derp.Wrap(err, location, "Receiving ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
