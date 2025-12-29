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

// GetSettings handles GET requests
func GetSettings(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return buildSettings(ctx, factory, session, user, build.ActionMethodGet)
}

// PostSettings handles POST/DELETE requests
func PostSettings(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return buildSettings(ctx, factory, session, user, build.ActionMethodPost)
}

// buildSettings is the common Settings handler for both GET and POST requests
func buildSettings(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User, actionMethod build.ActionMethod) error {

	const location = "handler.buildSettings"

	// Try to load the User's Inbox
	actionID := first.String(ctx.Param("action"), "general")

	if ok, err := handleJSONLD(ctx, user); ok {
		return derp.Wrap(err, location, "Unable to build JSON-LD")
	}

	builder, err := build.NewSettings(factory, session, ctx.Request(), ctx.Response(), user, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to create builder")
	}

	// Forward to the standard page builder to complete the job
	return build.AsHTML(ctx, factory, builder, actionMethod)
}
