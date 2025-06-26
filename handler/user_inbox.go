package handler

import (
	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
)

// GetInbox handles GET requests
func GetInbox(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return buildInbox(ctx, factory, user, build.ActionMethodGet)
}

// PostInbox handles POST/DELETE requests
func PostInbox(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return buildInbox(ctx, factory, user, build.ActionMethodPost)
}

// buildInbox is the common Inbox handler for both GET and POST requests
func buildInbox(ctx *steranko.Context, factory *domain.Factory, user *model.User, actionMethod build.ActionMethod) error {

	const location = "handler.buildInbox"

	// Try to load the User's Inbox
	actionID := first.String(ctx.Param("action"), "inbox")

	if ok, err := handleJSONLD(ctx, user); ok {
		return derp.Wrap(err, location, "Error building JSON-LD")
	}

	builder, err := build.NewInbox(factory, ctx.Request(), ctx.Response(), user, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Error creating builder")
	}

	// Forward to the standard page builder to complete the job
	return build.AsHTML(factory, ctx, builder, actionMethod)
}
