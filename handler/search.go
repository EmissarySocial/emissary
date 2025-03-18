package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/benpate/turbine/queue"
	"github.com/davecgh/go-spew/spew"
)

// IndexAllStreams is a handler function that triggers the IndexAllStreams queue task.
// It can only be called by an authenticated administrator.
func IndexAllStreams(ctx *steranko.Context, factory *domain.Factory) error {

	// Verify that this is an Administrator
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return derp.NewForbiddenError("handler.IndexAllStreams", "Only administrators can call this method")
	}

	// Create the Index task
	task := queue.NewTask("IndexAllStreams", mapof.Any{
		"host": ctx.Request().Host,
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
func IndexAllUsers(ctx *steranko.Context, factory *domain.Factory) error {

	// Verify that this is an Administrator
	authorization := getAuthorization(ctx)

	if !authorization.DomainOwner {
		return derp.NewForbiddenError("handler.IndexAllUsers", "Only administrators can call this method")
	}

	// Create the Index task
	task := queue.NewTask("IndexAllUsers", mapof.Any{
		"host": ctx.Request().Host,
	})

	// Execute the task in the background
	if err := factory.Queue().Publish(task); err != nil {
		return derp.Wrap(err, "handler.IndexAllUsers", "Error publishing task")
	}

	// Success.
	return ctx.NoContent(http.StatusOK)
}

func PostSearchLookup(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.PostSearchLookup"

	// Load the Stream from the database
	searchQueryService := factory.SearchQuery()
	searchQuery, err := searchQueryService.LoadOrCreate(ctx.Request().URL.Query())

	if err != nil {
		return derp.Wrap(err, location, "Error creating search query token")
	}

	forward := ctx.QueryParam("forward") + searchQueryService.ActivityPubURL(searchQuery.SearchQueryID)

	spew.Dump("PostSearchLookup ---------------------", ctx.Request().URL.Query(), searchQuery, forward)

	// Redirect to the new location, using a GET request.
	return ctx.Redirect(http.StatusSeeOther, forward)
}
