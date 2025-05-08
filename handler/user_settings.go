package handler

import (
	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
)

// GetSettings handles GET requests
func GetSettings(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return buildSettings(ctx, factory, user, build.ActionMethodGet)
}

// PostSettings handles POST/DELETE requests
func PostSettings(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return buildSettings(ctx, factory, user, build.ActionMethodPost)
}

// buildSettings is the common Settings handler for both GET and POST requests
func buildSettings(ctx *steranko.Context, factory *domain.Factory, user *model.User, actionMethod build.ActionMethod) error {

	const location = "handler.buildInbox"

	// Try to load the User's Inbox
	actionID := first.String(ctx.Param("action"), "inbox")

	if ok, err := handleJSONLD(ctx, user); ok {
		return derp.Wrap(err, location, "Error building JSON-LD")
	}

	builder, err := build.NewInbox(factory, ctx.Request(), ctx.Response(), user, actionID)

	if err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Error creating builder"))
	}

	// Forward to the standard page builder to complete the job
	return build.AsHTML(factory, ctx, builder, actionMethod)
}
