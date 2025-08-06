package consumer

import (
	"time"

	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/rosetta/channel"
	"github.com/rs/zerolog/log"
)

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

		if recordsFound := service.run(); !recordsFound {
			// For development purposes, there's only a short delay for localhost.
			if dt.IsLocalhost(service.host) {
				time.Sleep(20 * time.Second)
				continue
			}

			// "Regular" domains have a delay meant for production systems
			time.Sleep(10 * time.Minute)
			continue
		}
	}
}

// Run executes the SearchNotifier, scanning new SearchResults as they are created,
// and sending notifications to all followers with saved queries that match.
func (service *SearchNotifier) run() bool {

	const location = "service.SearchNotifier.Run"

	log.Trace().Msg("SearchNotifier: Scanning for new search results...")

	session, cancel, err := service.factory.Session(30 * time.Second)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error connecting to database"))
		return false
	}

	defer cancel()

	// Get the next batch of results
	resultsToNotify, err := service.searchResultService.GetResultsToNotify(session, service.processID)

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error getting locked results"))
		return false
	}

	// If there are no results, then wait before trying again.
	if len(resultsToNotify) == 0 {
		return false
	}

	log.Trace().Msgf("SearchNotifier: Found %v records", len(resultsToNotify))

	// Otherwise notify all global search followers
	if err := service.sendGlobalNotifications(resultsToNotify); err != nil {
		derp.Report(derp.Wrap(err, location, "Error sending notifications"))
		return true
	}

	// Then scan all saved search queries and send notifications
	if err := service.sendNotifications(session, resultsToNotify); err != nil {
		derp.Report(derp.Wrap(err, location, "Error sending notifications"))
		return true
	}

	// Last, mark all search results as "notified"
	if err := service.markNotified(session, resultsToNotify); err != nil {
		derp.Report(derp.Wrap(err, location, "Error sending notifications"))
		return true
	}

	return true
}
