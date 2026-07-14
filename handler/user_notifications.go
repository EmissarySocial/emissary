package handler

import (
	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
)

// GetNotifications handles GET requests
func GetNotifications(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return buildNotifications(ctx, factory, session, user, build.ActionMethodGet)
}

// PostNotifications handles POST/DELETE requests
func PostNotifications(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return buildNotifications(ctx, factory, session, user, build.ActionMethodPost)
}

// buildNotifications is the common Notifications handler for both GET and POST requests
func buildNotifications(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User, actionMethod build.ActionMethod) error {

	const location = "handler.buildNotifications"

	// Try to load the User's Notifications
	actionID := first.String(ctx.Param("action"), "index")

	if ok, err := handleJSONLD(ctx, user); ok {
		return derp.WrapIF(err, location, "Unable to build JSON-LD")
	}

	builder, err := build.NewNotifications(factory, session, ctx.Request(), ctx.Response(), user, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to create builder")
	}

	// Forward to the standard page builder to complete the job
	return build.AsHTML(ctx, factory, builder, actionMethod)
}
