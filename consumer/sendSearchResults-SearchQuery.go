package consumer

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SendSearchResult_SearchQuery(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendSearchResult_SearchQuery"

	// Collect services to use
	followerService := factory.Follower()
	queueService := factory.Queue()

	// Parse URL
	url := args.GetString("url")

	if url == "" {
		return queue.Failure(derp.InternalError(location, "'url' is required."))
	}

	// Parse SearchQueryID
	searchQueryID, err := primitive.ObjectIDFromHex(args.GetString("searchQueryId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "'searchQueryId' must be a valid ObjectID"))
	}

	// Calculate the ActorURL
	searchQueryService := factory.SearchQuery()
	actorURL := searchQueryService.ActivityPubURL(searchQueryID)

	// Get all Followers from the database
	followers := followerService.RangeBySearch(session, searchQueryID)

	// Send ActivityPub messages to each follower
	for follower := range followers {

		// Create a new queue message for each follower
		queueService.NewTask(
			"SendActivityPubMessage",
			mapof.Any{
				"host":      factory.Hostname(),
				"actorType": model.FollowerTypeSearch,
				"actorID":   searchQueryID.Hex(),
				"to":        follower.Actor.ProfileURL,
				"message": mapof.Any{
					vocab.PropertyActor:  actorURL,
					vocab.PropertyType:   vocab.ActivityTypeAnnounce,
					vocab.PropertyObject: url,
				},
			},
			queue.WithPriority(256),
		)
	}

	// Woot woot!
	return queue.Success()
}
