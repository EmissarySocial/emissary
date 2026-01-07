package activitypub_user

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
)

// outbox_CreateNote creates a new note (now, just a private message) in the user's outbox
func init() {

	outboxRouter.Add(vocab.ActivityTypeCreate, vocab.Any,

		func(context Context, document streams.Document) error {

			const location = "handler.activitypub_user.outbox_CreateNote"

			// Calculate all recipients of this Activity
			recipients := document.Recipients()

			// For now, we don't support public notes, so return an error
			// In the future, we'll add more rules that map public-facing posts to Streams.
			if recipients.Contains(vocab.NamespaceASPublic) {
				return derp.NotImplemented(location, "Public notes are not supported at this time.")
			}

			// Collect services
			factory := context.factory
			session := context.session
			locatorService := factory.Locator()
			outbox2Service := factory.Outbox2()
			objectService := factory.Object()

			// Find the local UserID
			actorID := document.Actor().ID()
			userID, err := locatorService.ParseUser(actorID)

			if err != nil {
				return derp.Wrap(err, location, "Unable to parse userID from actorID", "actorID: "+actorID)
			}

			if document.AtContext().IsNil() {
				document.SetProperty(vocab.AtContext, vocab.ContextTypeActivityStreams)
			}

			if document.Object().AtContext().IsNil() {
				document.Object().SetProperty(vocab.AtContext, vocab.ContextTypeActivityStreams)
			}

			// Create an Object record from this Activity
			object := model.NewObject()
			object.UserID = userID
			object.Value = document.Object().Map()
			object.Value.SetString(vocab.PropertyID, locatorService.ObjectURL(userID, object.ObjectID))
			object.Value.SetString(vocab.PropertyAttributedTo, actorID)
			object.Permissions = recipients

			if err := objectService.Save(session, &object, "Created via ActivityPub Outbox2"); err != nil {
				return derp.Wrap(err, location, "Unable to save Object from activity", document)
			}

			// Add an activity record to the Outbox2
			activity := model.NewActivity()
			activity.URL = locatorService.ActivityURL(model.ActorTypeUser, userID, activity.ActivityID)
			activity.ActorType = model.ActorTypeUser
			activity.ActorID = userID
			activity.Object = document.Map()
			activity.Recipients = recipients

			// Save the activity in the user's outbox
			if err := outbox2Service.Save(context.session, &activity, "Created via ActivityPub Outbox2"); err != nil {
				return derp.Wrap(err, location, "Unable to save Outbox2 activity")
			}

			// Done.
			context.context.Response().Header().Set("Location", locatorService.ObjectURL(userID, object.ObjectID))
			return context.context.NoContent(http.StatusCreated)
		},
	)
}
