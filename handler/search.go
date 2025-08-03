package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/benpate/turbine/queue"
)

// IndexAllStreams is a handler function that triggers the IndexAllStreams queue task.
// It can only be called by an authenticated administrator.
func IndexAllStreams(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

	// Verify that this is an Administrator
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return derp.ForbiddenError("handler.IndexAllStreams", "Only administrators can call this method")
	}

	// Create the Index task
	task := queue.NewTask("IndexAllStreams", mapof.Any{
		"host": dt.Hostname(ctx.Request()),
	})

	// Execute the task in the background
	if err := factory.Queue().Publish(task); err != nil {
		return derp.Wrap(err, "handler.IndexAllStreams", "Error publishing task")
	}

	// Success.
	return ctx.NoContent(http.StatusOK)
}

// IndexAllUsers is a handler function that triggers the IndexAllUsers queue task.
// It can only be called by an authenticated administrator.
func IndexAllUsers(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

	// Verify that this is an Administrator
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return derp.ForbiddenError("handler.IndexAllUsers", "Only administrators can call this method")
	}

	// Create the Index task
	task := queue.NewTask("IndexAllUsers", mapof.Any{
		"host": dt.Hostname(ctx.Request()),
	})

	// Execute the task in the background
	if err := factory.Queue().Publish(task); err != nil {
		return derp.Wrap(err, "handler.IndexAllUsers", "Error publishing task")
	}

	// Success.
	return ctx.NoContent(http.StatusOK)
}

func PostSearchLookup(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

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
