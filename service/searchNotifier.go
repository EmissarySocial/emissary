package service

import (
	"context"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// SearchNotifier
type SearchNotifier struct {
	searchQueryService  *SearchQuery
	searchResultService *SearchResult

	incoming <-chan model.SearchResult
	queued   []model.SearchResult
	working  []model.SearchResult
	timer    time.Timer
	context  context.Context
	mutex    sync.Mutex
}

func (service *SearchNotifier) Queue(result model.SearchResult) {
	service.mutex.Lock()
	service.queued = append(service.queued, result)
	service.mutex.Unlock()
}

func (service *SearchNotifier) Send() error {

	const location = "service.SearchNotifier.Send"

	// Claim the next batch of results
	service.mutex.Lock()
	service.working = service.queued
	service.queued = make([]model.SearchResult, 0)
	service.mutex.Unlock()

	if len(service.working) == 0 {
		return nil
	}

	// Get all search queries
	searchQueries, err := service.searchQueryService.RangeAll()

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving all search queries")
	}

	// Compare results with each search query
	for searchQuery := range searchQueries {

		for _, searchResult := range service.working {

			if searchQuery.Match(searchResult) {
				// Send the result to the user
			}
		}
	}

	return nil
}
