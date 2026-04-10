package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// ReceiveActivityPubAdd processes an incoming ActivityPub Add activity
// This is used to backfill the context of discussion threads when a new Object is
// Added to a Conversation Context (FEP-7888)
func ReceiveActivityPubAdd(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.ReceiveActivityPubMove"

	// Collect arguments
	actorID := args.GetString("actor")
	objectID := args.GetString("object")
	contextID := args.GetString("target")

	// Get an ActivityStream client
	client := factory.ActivityStream().AppClient()

	// Load the Object document to confirm it exists
	object, err := client.Load(objectID)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load Object document", "object: "+objectID))
	}

	if object.Context() != contextID {
		return queue.Failure(derp.BadRequest(location, "Object must be a member of the target context"))
	}

	// Load the context collection
	context, err := client.Load(contextID)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to load Context document", "context: "+contextID))
	}

	// RULE: The context document must be AttributedTo the provided ActorID
	if context.AttributedTo().ID() != actorID {
		return queue.Failure(derp.BadRequest(location, "Context must be attributed to the provided ActorID", "context: "+contextID, "actor: "+actorID))
	}

	// RULE: The context document must be a Collection
	if !context.IsCollection() {
		return queue.Failure(derp.BadRequest(location, "Context document must be a Collection", "context: "+contextID))
	}

	// Scan the collection for documents
	for document := range collections.RangeDocuments(context) {

		// Try to load the complete document
		document, err := client.Load(document.ID())

		if err != nil {
			return queue.Error(derp.Wrap(err, location, "Unable to load Document", "document: "+document.ID()))
		}

		// If this document was already in the cache, then we have successfully backfilled the context
		if document.HTTPHeader().Get(ascache.HeaderHannibalCache) == "true" {
			return queue.Success()
		}
	}

	// Ohio.
	return queue.Success()
}
