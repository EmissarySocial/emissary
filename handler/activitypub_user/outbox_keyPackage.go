package activitypub_user

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

func init() {
	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.ObjectTypeKeyPackage, outbox_CreateKeyPackage)
	outboxRouter.Add(vocab.ActivityTypeUpdate, vocab.ObjectTypeKeyPackage, outbox_UpdateKeyPackage)
	outboxRouter.Add(vocab.ActivityTypeAdd, vocab.ObjectTypeKeyPackage, outbox_AddKeyPackage)
	outboxRouter.Add(vocab.ActivityTypeRemove, vocab.ObjectTypeKeyPackage, outbox_RemoveKeyPackage)
	outboxRouter.Add(vocab.ActivityTypeDelete, vocab.ObjectTypeKeyPackage, outbox_DeleteKeyPackage)
}

// Create a new KeyPackage record from the ActivityPub API
func outbox_CreateKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_CreateKeyPackage"

	// RULE: Verify that the Domain allows MLS messages for this User
	domain := context.factory.Domain().Get()
	if !domain.UserCanMLS(context.user) {
		return derp.Forbidden(location, "MLS messages not allowed for this User")
	}

	// Extract the KeyPackage object from the Activity
	object := activity.Object()

	// RULE: The object must be attributed to the actor
	if object.AttributedTo().ID() != activity.Actor().ID() {
		return derp.Forbidden(location, "KeyPackage must be attributed to the actor", activity.Value())
	}

	// Populate the new KeyPackage
	keyPackageService := context.factory.MLSKeyPackage()
	keyPackage := model.NewKeyPackage()

	keyPackage.UserID = context.user.UserID
	keyPackage.MediaType = object.MediaType()
	keyPackage.Encoding = object.Encoding()
	keyPackage.Content = object.Content()
	keyPackage.GeneratorID = object.Generator().ID()
	keyPackage.GeneratorName = object.Generator().Name()

	// Save the KeyPackage to the database
	if err := keyPackageService.Save(context.session, &keyPackage, "Created via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to save KeyPackage")
	}

	// Write the response to the context
	context.context.Response().Header().Set("Location", keyPackageService.ActivityPubURL(keyPackage.UserID, keyPackage.KeyPackageID))
	return context.context.NoContent(http.StatusCreated)
}

// Update an existing KeyPackage record from the ActivityPub API
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
	if object.AttributedTo().ID() != activity.Actor().ID() {
		return derp.Forbidden(location, "KeyPackage must be attributed to the actor", activity.Value())
	}

	// Locate the existing KeyPackage
	keyPackageService := context.factory.MLSKeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByURL(context.session, object.ID(), &keyPackage); err != nil {
		return derp.Wrap(err, location, "Unable to load KeyPackage", "url", object.ID())
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
	keyPackage.GeneratorName = object.Generator().Name()

	// Save the KeyPackage to the database
	if err := keyPackageService.Save(context.session, &keyPackage, "Created via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to save KeyPackage")
	}

	// Write the response to the context
	context.context.Response().Header().Set("Location", keyPackageService.ActivityPubURL(keyPackage.UserID, keyPackage.KeyPackageID))
	return context.context.NoContent(http.StatusCreated)
}

// Locate and delete the KeyPackage referenced in the ActivityPub request
func outbox_DeleteKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_DeleteKeyPackage"

	// RULE: Verify that the Domain allows MLS messages for this User
	domain := context.factory.Domain().Get()
	if !domain.UserCanMLS(context.user) {
		return derp.Forbidden(location, "MLS messages not allowed for this User")
	}

	actor := activity.Actor()   // nolint:scopeguard
	object := activity.Object() // nolint:scopeguard

	// RULE: The actor must own the keyPackage
	if !strings.HasPrefix(object.ID(), actor.ID()) {
		return derp.Forbidden(location, "KeyPackage must be owned by this actor")
	}

	// Try to load the KeyPackage
	keyPackageService := context.factory.MLSKeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByURL(context.session, object.ID(), &keyPackage); err != nil {

		// If the KeyPackage doesn't exist, then "I have already won..."
		if derp.IsNotFound(err) {
			return nil
		}

		// Otherwise, you suck. I won't delete this KeyPackage for you.
		return derp.Wrap(err, location, "Unable to load KeyPackage", "url", object.ID())
	}

	// RULE: The actor must own the keyPackage
	if keyPackage.UserID != context.user.UserID {
		return derp.Forbidden(location, "KeyPackage must be owned by this actor")
	}

	// Delete the KeyPackage
	if err := keyPackageService.Delete(context.session, &keyPackage, "Deleted via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to delete KeyPackage")
	}

	// Win.
	return nil
}

// Add a KeyPackage to the user's collection (make it public)
func outbox_AddKeyPackage(context Context, activity streams.Document) error {
	return nil
}

// Remove a KeyPackage from the user's collection (make it private)
func outbox_RemoveKeyPackage(context Context, activity streams.Document) error {
	return nil
}
