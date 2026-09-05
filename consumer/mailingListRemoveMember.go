package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MailingListRemoveMember unsubscribes a departed Follower's address from their User's mailing list
func MailingListRemoveMember(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.MailingListRemoveMember"

	userID, err := primitive.ObjectIDFromHex(args.GetString("userId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid userId", args))
	}

	// RULE: the address comes from the arguments and cannot come from anywhere else. The
	// Follower was deleted before this task was released, so there is no record to read it
	// from -- which is why the enqueue writes it down.
	emailAddress := args.GetString("emailAddress")

	if emailAddress == "" {
		return queue.Failure(derp.BadRequest(location, "Email address is required", args))
	}

	if err := factory.UserConnection().MailchimpRemoveMember(session, userID, emailAddress); err != nil {
		return requeue(derp.Wrap(err, location, "Removing member from mailing list", args))
	}

	// Hasta la vista, baby
	return queue.Success()
}
