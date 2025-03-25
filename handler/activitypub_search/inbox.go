package activitypub_search

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/inbox"
	"github.com/benpate/steranko"
)

func PostInbox(ctx *steranko.Context, factory *domain.Factory, template *model.Template, stream *model.Stream, searchQuery *model.SearchQuery) error {

	const location = "handler.activitypub_search.PostInbox"

	// Retrieve the activity from the request body
	activity, err := inbox.ReceiveRequest(ctx.Request(), factory.ActivityStream())

	if err != nil {
		return derp.Wrap(err, location, "Error parsing ActivityPub request")
	}

	// Create a new request context for the ActivityPub router
	context := Context{
		factory:     factory,
		stream:      stream,
		searchQuery: searchQuery,
	}

	// Handle the ActivityPub request
	if err := inboxRouter.Handle(context, activity); err != nil {
		return derp.Wrap(err, location, "Error handling ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
