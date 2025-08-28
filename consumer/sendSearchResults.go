package consumer

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SendSearchResults scans for new SearchResults as they are created,
// and sends notifications to all followers with saved queries that match.
func SendSearchResults(factory *service.Factory, session data.Session, args mapof.Any) queue.Result {

	const location = "consumer.SendSearchResults"

	log.Trace().Msg("SendSearchResults: Scanning for new search results...")

	searchNotifierService := factory.SearchNotifier()
	searchResultService := factory.SearchResult()
	lockID := primitive.NewObjectID()

	// Get the next batch of results
	resultsToNotify, err := searchResultService.GetResultsToNotify(session, lockID)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error getting locked results"))
	}

	// If there are no results, then wait before trying again.
	if len(resultsToNotify) == 0 {
		return queue.Success()
	}

	log.Trace().Msgf("SendSearchResults: Found %v records", len(resultsToNotify))

	// Otherwise notify all global search followers
	if err := searchNotifierService.SendGlobalNotifications(resultsToNotify); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error sending notifications"))
	}

	// Then scan all saved search queries and send notifications
	if err := searchNotifierService.SendNotifications(session, resultsToNotify); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error sending notifications"))
	}

	// Last, mark all search results as "notified"
	if err := searchNotifierService.MarkNotified(session, resultsToNotify); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error sending notifications"))
	}

	return queue.Success()
}
