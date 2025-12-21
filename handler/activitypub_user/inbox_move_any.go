package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	inboxRouter.Add(vocab.ActivityTypeMove, vocab.Any, MoveAny)
}

func MoveAny(context Context, document streams.Document) error {

	// Locate/Move local actors
	locator := context.factory.Locator()

	if objectType, objectID, err := locator.GetObjectFromURL(context.session, document.Target().ID()); err == nil {

		if objectType == model.ActorTypeUser {
			return moveLocalUser(context, document, objectID)
		}
	}

	// For all other remote objects, schedule a background task
	context.factory.Queue().NewTask("ReceiveActivityPub-Move", mapof.Any{
		"actor":  document.Actor().ID(),
		"object": document.Object().ID(),
		"target": document.Target().ID(),
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
		return derp.Wrap(err, location, "Unable to load User", "userID", userID)
	}

	// Locate the Import record for this user
	importService := context.factory.Import()
	record := model.NewImport()

	if err := importService.LoadBySourceURL(context.session, user.UserID, document.Actor().ID(), &record); err != nil {
		return derp.Wrap(err, location, "Unable to load Import record", "userID", user.UserID, "sourceID", document.Actor().ID())
	}

	// RULE: Do not allow `Move` if the record is not in REVIEWING state
	if record.StateID != model.ImportStateReviewing {
		return derp.BadRequest(location, "Import record must be in REVIEWING state to accept a `Move` request.")
	}

	// Set Import record to "DO-MOVE" state
	// Remaining business logic is handled in service.doMove() method
	record.StateID = model.ImportStateDoMove

	if err := importService.Save(context.session, &record, "Finalizing Import"); err != nil {
		return derp.Wrap(err, location, "Unable to save Import record")
	}

	// Success let the client know that we've got it.
	return context.context.NoContent(http.StatusOK)
}
