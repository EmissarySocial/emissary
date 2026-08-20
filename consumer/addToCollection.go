package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AddToCollection adds a message to a collection / reply chain managed by this server.
func AddToCollection(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.AddToCollection"

	// Parse the UserID
	userID, err := primitive.ObjectIDFromHex(args.GetString("userId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Parsing userID", args))
	}

	// Parse the target URL to MAYBE add to a collection
	objectID := args.GetString("url")
	actorID := args.GetString("actorId")

	// Load/cache the document being added to the collection
	activityService := factory.ActivityStream()
	client := activityService.UserClient(userID)
	document, err := client.Load(objectID)

	if err != nil {
		return requeue(derp.Wrap(err, location, "Loading document", "url: "+objectID))
	}

	// Record this reply in the LOCAL parent's Replies collection, if the parent
	// is a Stream we own. This is independent of context ownership: we track
	// replies to our own Streams even when the thread's context is remote.
	if err := factory.Stream().AddReply(session, document.InReplyTo().String(), document.ID()); err != nil {
		return requeue(derp.Wrap(err, location, "Adding reply to parent collection", "url: "+objectID))
	}

	// Try to use the document's `context` to add it to a Collection.
	if result := addToCollection_Context(factory, session, actorID, document); result.NotNil() {
		return result
	}

	// Try to use the document's `inReplyTo` to add it to a Collection.
	if result := addToCollection_InReplyTo(factory, session, actorID, document); result.NotNil() {
		return result
	}

	// If we're here, then the document can't be added to a collection.
	derp.Report(derp.Internal(location, "Document could not be added to a collection", "url: "+objectID, document.Map()))

	// Return "success" to indicate that we don't need to requeue this task.
	return queue.Success()
}

// addToCollection_Context adds a document to the Collection named by its own "context" property
func addToCollection_Context(factory *service.Factory, session data.Session, actorID string, document streams.Document) queue.Result {

	// Get the context defined in the document.
	context := document.Context()

	if context == "" {
		return queue.Result{}
	}

	return addToCollection_Continue(factory, session, actorID, context, document.ID(), document.InReplyTo().String())
}

// addToCollection_InReplyTo adds a document to the Collection named by the nearest ancestor that has a "context"
func addToCollection_InReplyTo(factory *service.Factory, session data.Session, actorID string, document streams.Document) queue.Result {

	// Traverse the reply tree to find the closest ancestor with a `context`
	context, err := findRootContext(document, 5)

	// A load failure partway up the tree is (probably) transient, so requeue
	// instead of silently concluding that the document has no context.
	if err != nil {
		return requeue(err)
	}

	if context == "" {
		return queue.Result{}
	}

	return addToCollection_Continue(factory, session, actorID, context, document.ID(), document.InReplyTo().String())
}

// addToCollection_Continue resolves a context URL into a local Collection, then adds the document to it
func addToCollection_Continue(factory *service.Factory, session data.Session, actorID string, contextID string, url string, inReplyTo string) queue.Result {

	const location = "consumer.addToCollection_Continue"

	// RULE: If the context is empty, then there's nothing to do.
	if contextID == "" {
		return queue.Result{}
	}

	// Parse the Collection from the URL
	locatorService := factory.Locator()
	userID, collectionID, err := locatorService.ParseCollection(contextID)

	// If the context cannot be parsed, then this is not a collection that we manage.
	if err != nil {
		return queue.Result{}
	}

	// Load the Collection from the database.  NotFound here is usually TRANSIENT: until
	// outbound sends are queued (POST-COMMIT-TASKS-DESIGN.md Phase 3), the sender fans out
	// synchronously while its own transaction is still open, so a context Collection minted
	// for a brand-new thread may not be committed yet when this task runs.  Return a
	// retryable Error — not Failure — so the add survives the race; a genuinely missing
	// Collection still dies after the queue's bounded retries.
	collectionService := factory.Collection()
	collection := model.NewCollection()

	if err := collectionService.LoadByID(session, userID, collectionID, &collection); err != nil {
		return queue.Error(derp.Wrap(err, location, "Loading collection", collectionID.Hex()))
	}

	// Check permissions to see if the actor is allowed to add to this collection
	if collection.NotWritable(actorID) {
		return queue.Failure(derp.Forbidden(location, "Actor does not have permission to add to this collection", "collectionID: "+collectionID.Hex(), "userID: "+userID.Hex(), "actorID: "+actorID))
	}

	// Create a new CollectionItem record
	if err := collectionService.AddItem(session, &collection, url, inReplyTo); err != nil {
		return queue.Error(derp.Wrap(err, location, "Adding item to collection", collectionID.Hex(), userID.Hex(), url))
	}

	// Woot.
	return queue.Success()
}

// findRootContext walks up the reply tree looking for a "context", giving up after "count" more hops
func findRootContext(document streams.Document, count int) (string, error) {

	const location = "consumer.findRootContext"

	// RULE: If this document has a context, then return it.
	if context := document.Context(); context != "" {
		return context, nil
	}

	// RULE: If we've reached the last iteration, exit here.
	if count == 0 {
		return "", nil
	}

	// Reach UP the reply tree
	inReplyTo := document.InReplyTo()

	// If no reply is defined, then abort
	if inReplyTo.IsNil() {
		return "", nil
	}

	// If the parent is a bare link, load it from its remote server.  Propagate any
	// load error so the caller can requeue instead of silently dropping the document.
	if inReplyTo.IsString() {

		parent, err := inReplyTo.Load()

		if err != nil {
			return "", derp.Wrap(err, location, "Loading parent document", "inReplyTo: "+inReplyTo.ID())
		}

		return findRootContext(parent, count-1)
	}

	// Otherwise the parent is already inlined, so recurse into it directly.
	return findRootContext(inReplyTo, count-1)
}
