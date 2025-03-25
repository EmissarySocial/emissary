package service

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/rosetta/channel"
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

	host      string
	processID primitive.ObjectID
	context   context.Context
	queue     *queue.Queue
}

func NewSearchNotifier() SearchNotifier {
	return SearchNotifier{
		processID: primitive.NewObjectID(),
	}
}

func (service *SearchNotifier) Refresh(searchDomainService *SearchDomain, searchResultService *SearchResult, searchQueryService *SearchQuery, queue *queue.Queue, host string, context context.Context) {

	service.searchDomainService = searchDomainService
	service.searchResultService = searchResultService
	service.searchQueryService = searchQueryService

	service.queue = queue
	service.host = host
	service.context = context
}

// Run executes the SearchNotifier, scanning new SearchResults as they are created,
// and sending notifications to all followers with saved queries that match.
func (service *SearchNotifier) Run() {

	const location = "service.SearchNotifier.Run"

	log.Debug().Msg("Starting SearchNotifier")

	for {

		log.Trace().Msg("SearchNotifier: Scanning for new search results...")

		// If the context is closed, then exit this function
		if channel.Closed(service.context.Done()) {
			return
		}

		// Get the next batch of results
		resultsToNotify, err := service.searchResultService.GetResultsToNotify(service.processID)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error getting locked results"))
		}

		// If there are no results, then wait before trying again.
		if len(resultsToNotify) == 0 {

			// For development purposes, there's only a short delay for localhost.
			if domain.IsLocalhost(service.host) {
				time.Sleep(20 * time.Second)
				continue
			}

			// "Regular" domains have a delay meant for production systems
			time.Sleep(10 * time.Minute)
			continue
		}

		log.Trace().Msgf("SearchNotifier: Found %v records", len(resultsToNotify))

		// Otherwise notify all global search followers
		if err := service.sendGlobalNotifications(resultsToNotify); err != nil {
			derp.Report(derp.Wrap(err, location, "Error sending notifications"))
		}

		// Then scan all saved search queries and send notifications
		if err := service.sendNotifications(resultsToNotify); err != nil {
			derp.Report(derp.Wrap(err, location, "Error sending notifications"))
		}

		// Last, mark all search results as "notified"
		if err := service.markNotified(resultsToNotify); err != nil {
			derp.Report(derp.Wrap(err, location, "Error sending notifications"))
		}
	}
}

// sendGlobalNotifications sends notifications to all Global Search followers
func (service *SearchNotifier) sendGlobalNotifications(searchResults []model.SearchResult) error {

	const location = "service.SearchNotifier.sendNotifications"

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
			task := queue.NewTask(
				"SendSearchResults-Global",
				mapof.Any{
					"host":  service.host,
					"actor": actorID,
					"url":   searchResult.URL,
				},
				queue.WithPriority(200),
			)

			if err := service.queue.Publish(task); err != nil {
				return derp.Wrap(err, location, "Error publishing task")
			}
		}
	}

	// Success!
	return nil
}

// sendNotifications scans all SearchQueries in the database and sends notifications
// for all that match the provided batch of SearchResults
func (service *SearchNotifier) sendNotifications(searchResults []model.SearchResult) error {

	const location = "service.SearchNotifier.sendNotifications"

	// If there are no search results, then don't load search queries
	if len(searchResults) == 0 {
		return nil
	}

	// Get all search queries
	searchQueries, err := service.searchQueryService.RangeAll()

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

				task := queue.NewTask(
					"SendSearchResults-Query",
					mapof.Any{
						"host":          service.host,
						"actor":         actorID,
						"searchQueryID": searchQuery.SearchQueryID,
						"url":           searchResult.URL,
					},
					queue.WithPriority(200),
				)

				if err := service.queue.Publish(task); err != nil {
					return derp.Wrap(err, location, "Error publishing task")
				}
			}
		}
	}

	// Success!
	return nil
}

// markNotified marks all of the provided SearchResult records as being notified as of
// the current epoch.  This prevents them from being sent as duplicates in the future.
func (service *SearchNotifier) markNotified(searchResults []model.SearchResult) error {

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

		if err := service.searchResultService.Save(&searchResult, "Sent notifications"); err != nil {
			return derp.Wrap(err, location, "Error saving search result")
		}
	}

	// Success!
	return nil
}
