package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SendSearchResult_SearchQuery federates a single SearchResult to the Followers of one SearchQuery
func SendSearchResult_SearchQuery(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendSearchResult_SearchQuery"

	// Collect services to use
	followerService := factory.Follower()
	queueService := factory.Queue()

	// Parse URL
	url := args.GetString("url")

	if url == "" {
		return queue.Failure(derp.Internal(location, "'url' is required."))
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

	// Send ActivityPub messages to each follower. The activity carries its recipient in `to`;
	// hannibal/sender resolves the inbox and SendLocator.Actor signs as the SearchQuery actor
	// (F1). Tenant routing uses the activity's `actor` host. See F5.
	for follower := range followers {

		postcommit.Publish(
			session,
			queueService,
			sender.OutboxSendToAllRecipients,
			mapof.Any{
				vocab.AtContext:      vocab.ContextTypeActivityStreams,
				vocab.PropertyTo:     []string{follower.Actor.ProfileURL},
				vocab.PropertyActor:  actorURL,
				vocab.PropertyType:   vocab.ActivityTypeAnnounce,
				vocab.PropertyObject: url,
			},
		)
	}

	// Woot woot!
	return queue.Success()
}
