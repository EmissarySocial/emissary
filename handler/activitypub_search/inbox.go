package activitypub_search

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

func PostInbox(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream, searchQuery *model.SearchQuery) error {

	const location = "handler.activitypub_search.PostInbox"

	// Get an ActivityStream service for the Search Domain
	client := factory.ActivityStream().SearchDomainClient()

	// Create a new request context for the ActivityPub router
	context := Context{
		factory:     factory,
		session:     session,
		stream:      stream,
		searchQuery: searchQuery,
	}

	// Retrieve the activity from the request body
	if err := inboxRouter.ReceiveAndHandle(context, ctx.Request(), client); err != nil {
		return derp.Wrap(err, location, "Unable to receive ActivityPub request")
	}

	// Send the response to the client
	return ctx.String(http.StatusOK, "")
}
