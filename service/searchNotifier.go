package service

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/channel"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SearchNotifier
type SearchNotifier struct {
	searchQueryService  *SearchQuery
	searchResultService *SearchResult

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

func (service *SearchNotifier) Refresh(host string, context context.Context, searchQueryService *SearchQuery, searchResultService *SearchResult, queue *queue.Queue) {
	service.searchQueryService = searchQueryService
	service.searchResultService = searchResultService
	service.context = context
	service.queue = queue
	service.host = host
}

// Run executes the SearchNotifier, scanning new SearchResults as they are created,
// and sending notifications to all followers with saved queries that match.
func (service *SearchNotifier) Run() {

	const location = "service.SearchNotifier.Run"

	for {

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
			time.Sleep(10 * time.Minute)
			continue
		}

		// Otherwise, scan all saved search queries and send notifications
		if err := service.sendNotifications(resultsToNotify); err != nil {
			derp.Report(derp.Wrap(err, location, "Error sending notifications"))
		}
	}
}

func (service *SearchNotifier) sendNotifications(searchResults []model.SearchResult) error {

	const location = "service.SearchNotifier.sendNotifications"

	if len(searchResults) == 0 {
		return nil
	}

	// Get all search queries
	searchQueries, err := service.searchQueryService.RangeAll()

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving all search queries")
	}

	// Compare results with each search query
	for searchQuery := range searchQueries {

		for _, searchResult := range searchResults {

			if searchQuery.Match(searchResult) {

				if err := service.sendNotification(searchQuery, searchResult); err != nil {
					return derp.Wrap(err, location, "Error publishing task")
				}
			}
		}
	}

	// Mark all SearchResults as notified
	for _, searchResult := range searchResults {

		searchResult.LockID = primitive.NilObjectID
		searchResult.NotifiedDate = time.Now().Unix()
		searchResult.TimeoutDate = time.Now().Unix()

		if err := service.searchResultService.Save(&searchResult, "Sent notifications"); err != nil {
			return derp.Wrap(err, location, "Error saving search result")
		}
	}

	// Success!
	return nil
}

func (service *SearchNotifier) sendNotification(searchQuery model.SearchQuery, searchResult model.SearchResult) error {

	const location = "service.SearchNotifier.sendNotification"

	args := mapof.Any{
		"host":          service.host,
		"searchQueryID": searchQuery.SearchQueryID,
		"url":           searchResult.URL,
	}

	task := queue.NewTask("SendSearchResults", args, queue.WithPriority(200))

	if err := service.queue.Publish(task); err != nil {
		return derp.Wrap(err, location, "Error publishing task")
	}

	return nil
}
