package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// AddSearchResult is a queue consumer that adds a new SearchResult to the database,
// and sends an ActivityPub message to all followers who are listening for the same set of tags.
func AddSearchResult(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.AddSearchResult"

	// Insert/Update the SearchResult in the database
	searchService := factory.Search()
	searchResult := searchService.UnmarshalMap(args)

	if err := searchService.Sync(searchResult); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error saving search result"))
	}

	// Get All Followers who match this SearchResult
	followerService := factory.Follower()
	followers, err := followerService.RangeByTags(searchResult.TagValues...)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error loading followers"))
	}

	// Send Task to the Queue for each Follower
	q := factory.Queue()
	actorID := factory.Domain().ActorID()

	for follower := range followers {

		task := queue.NewTask("SendActivityPubMessage", mapof.Any{
			"actorType": model.FollowerTypeSearch,
			"inboxURL":  follower.Actor.InboxURL,
			"message": mapof.Any{
				vocab.AtContext:      vocab.ContextTypeActivityStreams,
				vocab.PropertyType:   vocab.ActivityTypeAnnounce,
				vocab.PropertyActor:  actorID,
				vocab.PropertyObject: searchResult.URL,
			},
		})

		if err := q.Publish(task); err != nil {
			return queue.Error(derp.Wrap(err, location, "Error sending message to queue"))
		}
	}

	return queue.Success()
}
