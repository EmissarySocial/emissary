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

// GetConversations handles GET requests
func GetConversations(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return buildConversations(ctx, factory, session, user, build.ActionMethodGet)
}

// PostConversations handles POST/DELETE requests
func PostConversations(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {
	return buildConversations(ctx, factory, session, user, build.ActionMethodPost)
}

// buildConversations is the common Conversations handler for both GET and POST requests
func buildConversations(ctx *steranko.Context, factory *service.Factory, session data.Session, user *model.User, actionMethod build.ActionMethod) error {

	const location = "handler.buildConversations"

	// Try to load the User's Conversations
	actionID := first.String(ctx.Param("action"), "index")

	if ok, err := handleJSONLD(ctx, user); ok {
		return derp.WrapIF(err, location, "Unable to build JSON-LD")
	}

	builder, err := build.NewConversations(factory, session, ctx.Request(), ctx.Response(), user, actionID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to create builder")
	}

	// Forward to the standard page builder to complete the job
	return build.AsHTML(ctx, factory, builder, actionMethod)
}
