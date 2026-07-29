package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the KeyPackage update handler with the outbox router
func init() {
	outboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeKeyPackage, outbox_UpdateKeyPackage)
}

// outbox_UpdateKeyPackage updates an existing KeyPackage record from the ActivityPub API
func outbox_UpdateKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_UpdateKeyPackage"

	// RULE: Verify that the Domain allows MLS messages for this User
	domain := context.factory.Domain().Get()
	if !domain.UserCanMLS(context.user) {
		return derp.Forbidden(location, "MLS messages not allowed for this User")
	}

	// Extract the KeyPackage object from the Activity
	object := activity.Object()

	// RULE: The object must be attributed to the actor
	if object.AttributedTo().ID() != activity.ActorID() {
		return derp.Forbidden(location, "KeyPackage must be attributed to the actor", activity.Value())
	}

	// Locate the existing KeyPackage
	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByURL(context.session, object.ID(), &keyPackage); err != nil {
		return derp.Wrap(err, location, "Loading KeyPackage", "url", object.ID())
	}

	// RULE: Guarantee that the KeyPackage belongs to the user making this request
	if keyPackage.UserID != context.user.UserID {
		return derp.Forbidden(location, "KeyPackage must be owned by this actor")
	}

	// RULE: Guarantee that the KeyPackage was created by the same Generator (device)
	if keyPackage.GeneratorID != object.Generator().ID() {
		return derp.Forbidden(location, "KeyPackage must be created by the same Generator")
	}

	// But you can update these values...
	keyPackage.MediaType = object.MediaType()
	keyPackage.Encoding = object.Encoding()
	keyPackage.Content = object.Content()
	keyPackage.Summary = object.Summary()
	keyPackage.GeneratorName = object.Generator().Name()

	// Save the KeyPackage to the database
	if err := keyPackageService.Save(context.session, &keyPackage, "Updated via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Saving KeyPackage")
	}
	// Update values in the activity object
	activity.SetProperty(vocab.PropertyObject, keyPackageService.ActivityPubURL(keyPackage.UserID, keyPackage.KeyPackageID))

	// Put the activity into the User's outbox (which triggers delivery to all recipients)
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Processing activity")
	}

	// Write the response to the context
	context.context.Response().Header().Set("Location", keyPackageService.ActivityPubURL(keyPackage.UserID, keyPackage.KeyPackageID))
	return context.context.NoContent(http.StatusAccepted)
}
