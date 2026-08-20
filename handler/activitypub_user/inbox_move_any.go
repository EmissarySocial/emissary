package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// init registers the handler for inbound Move activities
func init() {
	inboxRouter.Add(vocab.ActivityTypeMove, vocab.Any, inbox_MoveAny)
}

// inbox_MoveAny re-points local Followings at an actor's new address
func inbox_MoveAny(context Context, document streams.Document) error {

	const location = "activitypub_user.inbox_MoveAny"

	// Locate/Move local actors
	locator := context.factory.Locator()

	if objectType, objectID, err := locator.GetObjectFromURL(context.session, document.Target().ID()); err == nil {

		if objectType == model.ActorTypeUser {

			if err := moveLocalUser(context, document, objectID); err != nil {
				return derp.Wrap(err, location, "Moving local User", "userID", objectID)
			}
		}
	}

	// For all other remote objects, schedule a background task (post-commit)
	postcommit.Publish(context.session, context.factory.Queue(), "ReceiveActivityPub-Move", mapof.Any{
		"hostname": context.factory.Hostname(),
		"actor":    document.ActorID(),
		"object":   document.Object().ID(),
		"target":   document.Target().ID(),
	})

	// We have "Accepted" your request. That's the best you'll get for now.
	return context.context.NoContent(http.StatusAccepted)
}

// moveLocalUser handle messages that Move a remote profile into a
// local User account.  To do this, we must first have an active Import record
func moveLocalUser(context Context, document streams.Document, userID primitive.ObjectID) error {

	const location = "activitypub_user.Inbox.Move.LocalUser"

	// Locate the User from the database
	userService := context.factory.User()
	user := model.NewUser()

	if err := userService.LoadByID(context.session, userID, &user); err != nil {
		return derp.Wrap(err, location, "Loading User", "userID", userID)
	}

	// RULE: if the User has blocked the actor they are moving FROM, do not finalize the import (R20).
	blocked, err := context.factory.Rule().IsActorBlocked(context.session, user.UserID, document)

	if err != nil {
		return derp.Wrap(err, location, "Checking block rules", document.ActorID())
	}

	if blocked {
		return context.context.NoContent(http.StatusOK)
	}

	// Locate the Import record for this user
	importService := context.factory.Import()
	record := model.NewImport()

	if err := importService.LoadBySourceURL(context.session, user.UserID, document.ActorID(), &record); err != nil {

		return derp.Wrap(err, location, "Loading Import record", "userID", user.UserID, "sourceID", document.ActorID())
	}

	// RULE: Do not allow `Move` if the record is not in REVIEWING state
	if record.StateID != model.ImportStateReviewing {

		return derp.BadRequest(location, "Import record must be in REVIEWING state to accept a `Move` request.")
	}

	// Set Import record to "DO-MOVE" state
	// Remaining business logic is handled in service.doMove() method
	record.StateID = model.ImportStateDoMove

	if err := importService.Save(context.session, &record, "Finalizing Import"); err != nil {

		return derp.Wrap(err, location, "Saving Import record")
	}

	// Success let the client know that we've got it.
	return context.context.NoContent(http.StatusOK)
}
