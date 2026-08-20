package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// IndexAllStreams is a handler function that triggers the IndexAllStreams queue task.
// It can only be called by an authenticated administrator.
func IndexAllStreams(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.IndexAllStreams"

	// Collect required services
	searchService := factory.SearchResult()
	streamService := factory.Stream()

	// Get a RangeFunc containing all Streams in the database
	streams, err := streamService.RangePublished(session)

	if err != nil {
		return derp.Wrap(err, location, "Retrieving Streams")
	}

	// Index each Stream in the range
	for stream := range streams {

		// Recompute Hashtags
		originalHashtags := stream.Hashtags // nolint:scopeguard (caching value about to be changed)
		streamService.CalculateTags(session, &stream)

		// If necessary, re-save the Stream
		if !slice.Equal(stream.Hashtags, originalHashtags) {
			if err := streamService.Save(session, &stream, "Updating Hashtags"); err != nil {
				derp.Report(derp.Wrap(err, location, "Saving Stream"))
			}
		}

		// Create a new SearchResult from the (updated?) Stream
		searchResult := streamService.SearchResult(&stream)

		if err := searchService.Sync(session, searchResult); err != nil {
			derp.Report(derp.Wrap(err, location, "Saving SearchResult"))
		}
	}

	// Success.
	return ctx.NoContent(http.StatusOK)
}

// IndexAllUsers is a handler function that triggers the IndexAllUsers queue task.
// It can only be called by an authenticated administrator.
func IndexAllUsers(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.IndexAllUsers"

	searchService := factory.SearchResult()
	userService := factory.User()

	allUsers, err := userService.RangeAll(session)

	if err != nil {
		return derp.Wrap(err, location, "Querying Users")
	}

	for user := range allUsers {

		// Recompute Hashtags
		originalHashtags := user.Hashtags // nolint:scopeguard (caching value about to be changed)
		userService.CalculateTags(session, &user)

		// If necessary, re-save the User (this also backfills the denormalized TagURL)
		if !slice.Equal(user.Hashtags, originalHashtags) {
			if err := userService.Save(session, &user, "Updating Hashtags"); err != nil {
				derp.Report(derp.Wrap(err, location, "Saving User"))
			}
		}

		// Create a new SearchResult from the (updated?) User
		searchResult := userService.SearchResult(&user)

		if err := searchService.Sync(session, searchResult); err != nil {
			derp.Report(derp.Wrap(err, location, "Saving SearchResult"))
		}
	}

	// Success.
	return ctx.NoContent(http.StatusOK)
}

// ReindexReplies re-projects all reply Streams into their parents' Replies collections
// and refreshes reply counts. Admin-only; safe to re-run.
func ReindexReplies(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.ReindexReplies"

	if err := factory.Stream().ReindexReplies(session); err != nil {
		return derp.Wrap(err, location, "Reindexing replies")
	}

	return ctx.NoContent(http.StatusOK)
}

// PostSearchLookup records a visitor's search selection, and is callable only from a page on this domain
func PostSearchLookup(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostSearchLookup"

	// Collect and validate the referer/URL
	referer := ctx.Request().Header.Get("referer")

	if referer == "" {
		return derp.Forbidden(location, "No referer", referer)
	}

	if uri.Hostname(referer) != factory.Hostname() {
		return derp.Forbidden(location, "Invalid referer", referer)
	}

	// Load the Stream from the database
	searchQueryService := factory.SearchQuery()
	searchQuery, err := searchQueryService.LoadOrCreate(session, ctx.QueryParams())

	if err != nil {
		return derp.Wrap(err, location, "Creating search query token")
	}

	// Set the referer/URL if it's not already set
	if searchQuery.URL == "" {
		searchQuery.URL = referer
		if err := searchQueryService.Save(session, &searchQuery, "Set source URL"); err != nil {
			return derp.Wrap(err, location, "Applying URL to search query")
		}
	}

	// Redirect to the new location, using a GET request.
	forward := ctx.QueryParam("forward") + searchQueryService.ActivityPubURL(searchQuery.SearchQueryID)
	return ctx.Redirect(http.StatusSeeOther, forward)
}
