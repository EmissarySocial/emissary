package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchNotifier
type SearchNotifier struct {
	searchDomainService *SearchDomain
	searchResultService *SearchResult
	searchQueryService  *SearchQuery

	host  string
	queue *queue.Queue
}

func NewSearchNotifier() SearchNotifier {
	return SearchNotifier{}
}

func (service *SearchNotifier) Refresh(searchDomainService *SearchDomain, searchResultService *SearchResult, searchQueryService *SearchQuery, queue *queue.Queue, host string) {
	service.searchDomainService = searchDomainService
	service.searchResultService = searchResultService
	service.searchQueryService = searchQueryService

	service.queue = queue
	service.host = host
}

// sendGlobalNotifications sends notifications to all Global Search followers
func (service *SearchNotifier) SendGlobalNotifications(searchResults []model.SearchResult) error {

	// If there are no search results, then don't load search queries
	if len(searchResults) == 0 {
		return nil
	}

	actorID := service.searchDomainService.ActivityPubURL()

	// Scan each SearchResult in our current batch...
	for _, searchResult := range searchResults {

		// Only notify on new LOCAL search results (don't syndicate results we got from other servers)
		if searchResult.Local {

			log.Trace().Str("URL", searchResult.URL).Msg("Sending global notification")

			// Send notifications to all followers
			service.queue.Enqueue <- queue.NewTask(
				"SendSearchResults-Global",
				mapof.Any{
					"host":  service.host,
					"actor": actorID,
					"url":   searchResult.URL,
				},
				queue.WithPriority(256),
			)
		}
	}

	// Success!
	return nil
}

// sendNotifications scans all SearchQueries in the database and sends notifications
// for all that match the provided batch of SearchResults
func (service *SearchNotifier) SendNotifications(session data.Session, searchResults []model.SearchResult) error {

	const location = "service.SearchNotifier.sendNotifications"

	// If there are no search results, then don't load search queries
	if len(searchResults) == 0 {
		return nil
	}

	// Get all search queries
	searchQueries, err := service.searchQueryService.RangeAll(session)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving all search queries")
	}

	// Scan each SearchQuery in the database...
	for searchQuery := range searchQueries {

		// Scan each SearchResult in our current batch...
		for _, searchResult := range searchResults {

			// Send notifications for any matches
			if searchQuery.Match(searchResult) {
				actorID := service.searchQueryService.ActivityPubURL(searchQuery.SearchQueryID)

				service.queue.Enqueue <- queue.NewTask(
					"SendSearchResults-Query",
					mapof.Any{
						"host":          service.host,
						"actor":         actorID,
						"searchQueryID": searchQuery.SearchQueryID,
						"url":           searchResult.URL,
					},
					queue.WithPriority(256),
				)
			}
		}
	}

	// Success!
	return nil
}

// markNotified marks all of the provided SearchResult records as being notified as of
// the current epoch.  This prevents them from being sent as duplicates in the future.
func (service *SearchNotifier) MarkNotified(session data.Session, searchResults []model.SearchResult) error {

	const location = "service.SearchNotifier.sendNotifications"

	// If there are no search results, then don't load search queries
	if len(searchResults) == 0 {
		return nil
	}

	// Mark all SearchResults as notified
	for _, searchResult := range searchResults {

		searchResult.NotifiedDate = time.Now().Unix()
		searchResult.LockID = primitive.NilObjectID
		searchResult.TimeoutDate = 0

		if err := service.searchResultService.Save(session, &searchResult, "Sent notifications"); err != nil {
			return derp.Wrap(err, location, "Error saving search result")
		}
	}

	// Success!
	return nil
}
