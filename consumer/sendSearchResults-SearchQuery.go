package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SendSearchResults(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.SendSearchResults"

	// Collect services to use
	followerService := factory.Follower()
	queueService := factory.Queue()

	// Parse URL
	url := args.GetString("url")

	if url == "" {
		return queue.Failure(derp.NewInternalError(location, "'url' is required."))
	}

	// Parse ActorID
	actorURL := args.GetString("actor")

	if actorURL == "" {
		return queue.Failure(derp.NewInternalError(location, "'actor' is required."))
	}

	// Parse SearchQueryID
	searchQueryID, err := primitive.ObjectIDFromHex(args.GetString("searchQueryID"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "'searchQueryID' must be a valid ObjectID"))
	}

	// Get all Followers from the database
	followers, err := followerService.RangeBySearch(searchQueryID)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error retrieving followers"))
	}

	// Send ActivityPub messages to each follower
	for follower := range followers {

		// Create a new queue message for each follower
		task := queue.NewTask(
			"SendActivityPubMessage",
			mapof.Any{
				"host":          factory.Hostname(),
				"actorType":     model.FollowerTypeSearch,
				"searchQueryID": searchQueryID.Hex(),
				"inboxURL":      follower.Actor.InboxURL,
				"message": mapof.Any{
					vocab.PropertyActor:  actorURL,
					vocab.PropertyType:   vocab.ActivityTypeAnnounce,
					vocab.PropertyObject: url,
				},
			},
			queue.WithPriority(200),
		)

		// Send the message to the queue
		if err := queueService.Publish(task); err != nil {
			return queue.Error(derp.Wrap(err, location, "Error sending message to queue"))
		}
	}

	// Woot woot!
	return queue.Success()
}
