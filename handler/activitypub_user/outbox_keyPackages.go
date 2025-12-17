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
	outboxRouter.Add(vocab.ActivityTypeAdd, vocab.ObjectTypeKeyPackage, outbox_AddKeyPackage)
	outboxRouter.Add(vocab.ActivityTypeRemove, vocab.ObjectTypeKeyPackage, outbox_RemoveKeyPackage)
	outboxRouter.Add(vocab.ActivityTypeDelete, vocab.ObjectTypeKeyPackage, outbox_DeleteKeyPackage)
}

// Create a new KeyPackage record from the ActivityPub API
func outbox_CreateKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_CreateKeyPackage"

	object := activity.Object()

	// RULE: The object must be attributed to the actor
	if object.AttributedTo().ID() != activity.Actor().ID() {
		return derp.ForbiddenError(location, "KeyPackage must be attributed to the actor", activity.Value())
	}

	// Populate the new KeyPackage
	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	keyPackage.UserID = context.user.UserID
	keyPackage.MediaType = object.MediaType()
	keyPackage.Encoding = object.Encoding()
	keyPackage.Content = object.Content()
	keyPackage.Generator = object.Generator().ID()

	// Save the KeyPackage to the database
	if err := keyPackageService.Save(context.session, &keyPackage, "Created via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to save KeyPackage")
	}

	// Write the response to the context
	if err := context.context.JSON(http.StatusCreated, keyPackageService.GetJSONLD(&keyPackage)); err != nil {
		return derp.Wrap(err, location, "Unable to send response")
	}

	// Success
	return nil
}

// Locate and delete the KeyPackage referenced in the ActivityPub request
func outbox_DeleteKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_DeleteKeyPackage"

	actor := activity.Actor()
	object := activity.Object()

	// RULE: The actor must own the keyPackage
	if !strings.HasPrefix(object.ID(), actor.ID()) {
		return derp.ForbiddenError(location, "KeyPackage must be owned by this actor")
	}

	// Load the KeyPackage
	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByURL(context.session, object.ID(), &keyPackage); err != nil {
		return derp.Wrap(err, location, "Unable to load KeyPackage", "url", object.ID())
	}

	// RULE: The actor must own the keyPackage
	if keyPackage.UserID != context.user.UserID {
		return derp.ForbiddenError(location, "KeyPackage must be owned by this actor")
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
	return outbox_SetKeyPackageVisibility(context, activity, true)
}

// Remove a KeyPackage from the user's collection (make it private)
func outbox_RemoveKeyPackage(context Context, activity streams.Document) error {
	return outbox_SetKeyPackageVisibility(context, activity, true)
}

// Set the visibility of a KeyPackage in the user's collection
func outbox_SetKeyPackageVisibility(context Context, activity streams.Document, isPublic bool) error {

	const location = "handler.activitypub_user.outbox_AddKeyPackage"

	// Collect values from the activity
	actor := activity.Actor()
	object := activity.Object()
	target := activity.Target()

	// RULE: The actor must own the target (keyPackage collection)
	if !strings.HasPrefix(target.ID(), actor.ID()) {
		return derp.ForbiddenError(location, "Target collection must be owned by this actor")
	}

	// RULE: The actor must own the keyPackage
	if !strings.HasPrefix(object.ID(), actor.ID()) {
		return derp.ForbiddenError(location, "KeyPackage must be owned by this actor")
	}

	// Load the KeyPackage
	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByURL(context.session, object.ID(), &keyPackage); err != nil {
		return derp.Wrap(err, location, "Unable to load KeyPackage", "url", object.ID())
	}

	// Set the visibility
	keyPackage.IsPublic = isPublic

	// Save the KeyPackage to the database
	if err := keyPackageService.Save(context.session, &keyPackage, "Published via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to save KeyPackage")
	}

	// Yup.
	return nil
}
