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

func SendSearchResult(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendSearchResult"

	// Validate SearchResultID argument
	searchResultID, err := primitive.ObjectIDFromHex(args.GetString("searchResultId"))

	if err != nil {
		return queue.Failure(derp.Wrap(err, location, "Invalid SearchResultID", args))
	}

	// Load the SearchResult
	searchResultService := factory.SearchResult()
	searchResult := model.NewSearchResult()

	if err := searchResultService.LoadByID(session, searchResultID, &searchResult); err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to retrieve SearchResult"))
	}

	// PART 1:
	// Send SearchResult to matching SearchQueries
	//

	// Find SearchQueries that are "near matches" to this result
	searchQueryService := factory.SearchQuery()
	searchQueries, err := searchQueryService.RangeNearMatches(session, &searchResult)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Unable to retrieve SearchQueries from database"))
	}

	// Let's get ready to rumble...
	queueService := factory.Queue()

	for searchQuery := range searchQueries {

		// if this SearchQuery ACTUALLY matches...
		if searchQuery.Match(&searchResult) {

			// Queue up a task to notify its followers
			queueService.NewTask(
				"SendSearchResult-SearchQuery",
				mapof.Any{
					"host":          factory.Hostname(),
					"activity":      args.GetString("activity"),
					"searchQueryId": searchQuery.SearchQueryID,
					"url":           searchResult.URL,
				},

				// Run immediately, if possible
				queue.WithPriority(32),
			)
		}
	}

	// PART 2:
	// Send SearchResult to all Global Search Followers
	//

	// Get all Followers from the database
	searchDomainService := factory.SearchDomain()
	followerService := factory.Follower()
	followers := followerService.RangeByGlobalSearch(session)

	// Send ActivityPub messages to each follower
	for follower := range followers {

		// Create a new queue message for each follower
		queueService.NewTask(
			"SendActivityPubMessage",
			mapof.Any{
				"host":      factory.Hostname(),
				"actorType": model.FollowerTypeSearchDomain,
				"actorID":   primitive.NilObjectID.Hex(),
				"to":        follower.Actor.ProfileURL,
				"message": mapof.Any{
					vocab.PropertyActor:  searchDomainService.ActivityPubURL(),
					vocab.PropertyType:   vocab.ActivityTypeAnnounce,
					vocab.PropertyObject: searchResult.URL,
				},
			},
			queue.WithPriority(256),
		)
	}

	// SUCCESS!!!
	return queue.Success()
}
