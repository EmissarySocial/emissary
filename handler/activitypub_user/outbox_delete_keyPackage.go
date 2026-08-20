package activitypub_user

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// init registers the handler for outbound Delete/KeyPackage activities
func init() {
	outboxRouter.Add(vocab.ActivityTypeDelete, vocab.ObjectTypeKeyPackage, outbox_DeleteKeyPackage)
}

// Locate and delete the KeyPackage referenced in the ActivityPub request
func outbox_DeleteKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_DeleteKeyPackage"

	// RULE: Verify that the Domain allows MLS messages for this User
	if domain := context.factory.Domain().Get(); !domain.UserCanMLS(context.user) {
		return derp.Forbidden(location, "MLS messages not allowed for this User")
	}

	actor := activity.Actor()   // nolint:scopeguard
	object := activity.Object() // nolint:scopeguard

	// RULE: The actor must own the keyPackage
	if !strings.HasPrefix(object.ID(), actor.ID()) {
		return derp.Forbidden(location, "KeyPackage must be owned by this actor")
	}

	// Try to load the KeyPackage
	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByURL(context.session, object.ID(), &keyPackage); err != nil {

		// If the KeyPackage doesn't exist, then "I have already won..."
		if derp.IsNotFound(err) {
			return nil
		}

		// Otherwise, you suck. I won't delete this KeyPackage for you.
		return derp.Wrap(err, location, "Loading KeyPackage", "url", object.ID())
	}

	// RULE: The actor must own the keyPackage
	if keyPackage.UserID != context.user.UserID {
		return derp.Forbidden(location, "KeyPackage must be owned by this actor")
	}

	// Delete the KeyPackage
	if err := keyPackageService.Delete(context.session, &keyPackage, "Deleted via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Deleting KeyPackage")
	}

	// Put the activity into the User's outbox (which triggers delivery to all recipients)
	if err := putActivityIntoOutbox(context, activity); err != nil {
		return derp.Wrap(err, location, "Processing activity")
	}

	// Win.
	return nil
}
