package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/benpate/turbine/queue"
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
