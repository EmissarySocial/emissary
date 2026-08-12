package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the KeyPackage activity handlers with the outbox router
func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeKeyPackage, outbox_CreateKeyPackage)
	outboxRouter.Add(vocab.ActivityTypeAdd, vocab.ObjectTypeKeyPackage, outbox_AddKeyPackage)
	outboxRouter.Add(vocab.ActivityTypeRemove, vocab.ObjectTypeKeyPackage, outbox_RemoveKeyPackage)
}

// outbox_CreateKeyPackage creates a new KeyPackage record from the ActivityPub API
func outbox_CreateKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_CreateKeyPackage"

	// RULE: Verify that the Domain allows MLS messages for this User
	if domain := context.factory.Domain().Get(); !domain.UserCanMLS(context.user) {
		return derp.Forbidden(location, "MLS messages not allowed for this User")
	}

	// Extract the KeyPackage object from the Activity
	object := activity.Object()

	// RULE: The object must be attributed to the actor
	if object.AttributedTo().ID() != activity.ActorID() {
		return derp.Forbidden(location, "KeyPackage must be attributed to the actor", activity.Value())
	}

	// Populate the new KeyPackage
	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	keyPackage.UserID = context.user.UserID
	keyPackage.MediaType = object.MediaType()
	keyPackage.Encoding = object.Encoding()
	keyPackage.Content = object.Content()
	keyPackage.Summary = object.Summary()
	keyPackage.GeneratorID = object.Generator().ID()
	keyPackage.GeneratorName = object.Generator().Name()
	keyPackage.Ciphersuite = object.MLSCiphersuite()

	// Save the KeyPackage to the database
	if err := keyPackageService.Save(context.session, &keyPackage, "Created via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Saving KeyPackage")
	}

	// Put the activity into the User's outbox (which triggers delivery to all recipients)
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Processing activity")
	}

	// Write the response to the context
	context.context.Response().Header().Set("Location", keyPackageService.ActivityPubURL(keyPackage.UserID, keyPackage.KeyPackageID))
	return context.context.NoContent(http.StatusCreated)
}

// outbox_AddKeyPackage adds a KeyPackage to the user's collection (make it public)
func outbox_AddKeyPackage(_ Context, _ streams.Document) error {
	return nil
}

// outbox_RemoveKeyPackage removes a KeyPackage from the user's collection (make it private)
func outbox_RemoveKeyPackage(_ Context, _ streams.Document) error {
	return nil
}
