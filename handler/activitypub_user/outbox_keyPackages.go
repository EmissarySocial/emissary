package activitypub_user

import (
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

// This funciton handles ActivityPub "Accept/Follow" activities, meaning that
// it is called with a remote server accepts our follow request.
func outbox_CreateKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_CreateKeyPackage"

	object := activity.Object()

	// RULE: The object must be attributed to the actor
	if object.AttributedTo().ID() != activity.Actor().ID() {
		return derp.ForbiddenError(location, "KeyPackage must be attributed to the actor")
	}

	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	keyPackage.UserID = context.user.UserID
	keyPackage.MediaType = object.MediaType()
	keyPackage.Encoding = object.Encoding()
	keyPackage.Content = object.Content()
	keyPackage.Generator = object.Generator().ID()

	if err := keyPackageService.Save(context.session, &keyPackage, "Created via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to save KeyPackage")
	}

	return nil
}

func outbox_AddKeyPackage(context Context, activity streams.Document) error {
	return outbox_SetKeyPackageVisibility(context, activity, true)
}

func outbox_RemoveKeyPackage(context Context, activity streams.Document) error {
	return outbox_SetKeyPackageVisibility(context, activity, true)
}

func outbox_DeleteKeyPackage(context Context, activity streams.Document) error {

	const location = "handler.activitypub_user.outbox_DeleteKeyPackage"

	actor := activity.Actor()
	object := activity.Object()

	// RULE: The actor must own the keyPackage
	if !strings.HasPrefix(object.ID(), actor.ID()) {
		return derp.ForbiddenError(location, "KeyPackage must be owned by this actor")
	}

	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByURL(context.session, object.ID(), &keyPackage); err != nil {
		return derp.Wrap(err, location, "Unable to load KeyPackage", "url", object.ID())
	}

	if err := keyPackageService.Delete(context.session, &keyPackage, "Deleted via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to delete KeyPackage")
	}

	return nil
}

func outbox_SetKeyPackageVisibility(context Context, activity streams.Document, isPublic bool) error {

	const location = "handler.activitypub_user.outbox_AddKeyPackage"

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

	keyPackageService := context.factory.KeyPackage()
	keyPackage := model.NewKeyPackage()

	if err := keyPackageService.LoadByURL(context.session, object.ID(), &keyPackage); err != nil {
		return derp.Wrap(err, location, "Unable to load KeyPackage", "url", object.ID())
	}

	keyPackage.IsPublic = isPublic

	if err := keyPackageService.Save(context.session, &keyPackage, "Published via ActivityPub API"); err != nil {
		return derp.Wrap(err, location, "Unable to save KeyPackage")
	}

	return nil
}
