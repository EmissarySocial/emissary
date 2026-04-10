package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// AddToContext adds a message to a context / reply chain managed by this server.
func AddToContext(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.AddToContext"

	// Collect parameters
	objectID := args.GetString("url")

	activityService := factory.ActivityStream()
	client := activityService.AppClient()

	// Load the document (probably from the cache)
	document, err := client.Load(objectID)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Unable to load document"))
	}

	// Create a new ObjectLink record
	objectLink := model.NewObjectLink()
	objectLink.Context = document.Context()
	objectLink.InReplyTo = document.InReplyTo().ID()
	objectLink.Object = document.ID()
	objectLink.Recipients = document.Recipients()

	// Save the unique ObjectLink record to the database
	if err := factory.Context().SaveUnique(session, &objectLink, "Created"); err != nil {
		return queue.Error(err)
	}

	// Woot.
	return queue.Success()
}
