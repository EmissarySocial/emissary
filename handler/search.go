package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
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
		return derp.Wrap(err, location, "Error retrieving Streams")
	}

	// Index each Stream in the range
	for stream := range streams {

		// Recompute Hashtags
		originalHashtags := stream.Hashtags
		streamService.CalculateTags(session, &stream)

		// If necessary, re-save the Stream
		if !slice.Equal(stream.Hashtags, originalHashtags) {
			if err := streamService.Save(session, &stream, "Updating Hashtags"); err != nil {
				derp.Report(derp.Wrap(err, location, "Error saving Stream"))
			}
		}

		// Create a new SearchResult from the (updated?) Stream
		searchResult := streamService.SearchResult(&stream)

		if err := searchService.Sync(session, searchResult); err != nil {
			derp.Report(derp.Wrap(err, location, "Error saving SearchResult"))
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
		return derp.Wrap(err, location, "Unable to query Users")
	}

	for user := range allUsers {

		searchResult := userService.SearchResult(&user)

		if err := searchService.Sync(session, searchResult); err != nil {
			derp.Report(derp.Wrap(err, location, "Unable to save SearchResult"))
		}
	}

	// Success.
	return ctx.NoContent(http.StatusOK)
}

func PostSearchLookup(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostSearchLookup"

	// Collect and validate the referer/URL
	referer := ctx.Request().Header.Get("referer")

	if referer == "" {
		return derp.ForbiddenError(location, "No referer", referer)
	}

	if dt.NameOnly(referer) != factory.Hostname() {
		return derp.ForbiddenError(location, "Invalid referer", referer)
	}

	// Load the Stream from the database
	searchQueryService := factory.SearchQuery()
	searchQuery, err := searchQueryService.LoadOrCreate(session, ctx.QueryParams())

	if err != nil {
		return derp.Wrap(err, location, "Error creating search query token")
	}

	// Set the referer/URL if it's not already set
	if searchQuery.URL == "" {
		searchQuery.URL = referer
		if err := searchQueryService.Save(session, &searchQuery, "Set source URL"); err != nil {
			return derp.Wrap(err, location, "Error applying URL to search query")
		}
	}

	// Redirect to the new location, using a GET request.
	forward := ctx.QueryParam("forward") + searchQueryService.ActivityPubURL(searchQuery.SearchQueryID)
	return ctx.Redirect(http.StatusSeeOther, forward)
}
