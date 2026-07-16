package consumer

import (
	"github.com/EmissarySocial/emissary/model"
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
		return queue.Error(derp.Wrap(err, location, "Retrieving SearchResult", args))
	}

	// PART 1:
	// Send SearchResult to matching SearchQueries
	//

	// Find SearchQueries that are "near matches" to this result
	searchQueryService := factory.SearchQuery()
	searchQueries, err := searchQueryService.RangeNearMatches(session, &searchResult)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Retrieving SearchQueries from database"))
	}

	// Let's get ready to rumble...
	queueService := factory.Queue()

	for searchQuery := range searchQueries {

		// if this SearchQuery ACTUALLY matches...
		if searchQuery.Match(&searchResult) {

			// Queue up a task (post-commit) to notify its followers
			postcommit.Publish(
				session,
				queueService,
				"SendSearchResult-SearchQuery",
				mapof.Any{
					"hostname":      factory.Hostname(),
					"searchQueryId": searchQuery.SearchQueryID,
					"url":           searchResult.URL,
				},
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

	// Send ActivityPub messages to each follower. The activity carries its recipient in `to`;
	// hannibal/sender resolves the inbox and SendLocator.Actor signs as the global @search actor
	// (F1). Tenant routing uses the activity's `actor` host. See F5.
	for follower := range followers {

		postcommit.Publish(
			session,
			queueService,
			sender.OutboxSendToAllRecipients,
			mapof.Any{
				vocab.AtContext:      vocab.ContextTypeActivityStreams,
				vocab.PropertyTo:     []string{follower.Actor.ProfileURL},
				vocab.PropertyActor:  searchDomainService.ActivityPubURL(),
				vocab.PropertyType:   vocab.ActivityTypeAnnounce,
				vocab.PropertyObject: searchResult.URL,
			},
		)
	}

	// SUCCESS!!!
	return queue.Success()
}
