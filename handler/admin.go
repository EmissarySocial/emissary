package handler

import (
	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetAdmin handles GET requests
func GetAdmin(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return buildAdmin(ctx, factory, session, build.ActionMethodGet)
}

// PostAdmin handles POST/DELETE requests
func PostAdmin(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	return buildAdmin(ctx, factory, session, build.ActionMethodPost)
}

func buildAdmin(ctx *steranko.Context, factory *service.Factory, session data.Session, actionMethod build.ActionMethod) error {

	const location = "handler.adminBuilder"

	if !isOwner(ctx.Authorization()) {
		return derp.Forbidden(location, "Unauthorized")
	}

	// Parse admin parameters
	templateID, actionID, objectID := buildAdmin_ParsePath(ctx)

	// Try to load the Template
	templateService := factory.Template()
	template, err := templateService.LoadAdmin(templateID)

	if err != nil {
		return err
	}

	// Locate and populate the builder
	builder, err := buildAdmin_GetBuilder(ctx, factory, session, template, actionID, objectID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to generate builder")
	}

	// Success!!
	return build.AsHTML(ctx, factory, builder, actionMethod)
}

func buildAdmin_ParsePath(ctx echo.Context) (string, string, primitive.ObjectID) {

	// First parameter is always the templateID
	templateID := first.String(ctx.Param("param1"), "domain")

	// If the second parameter is an ObjectID, then we parse object/action
	if objectID, err := primitive.ObjectIDFromHex(ctx.Param("param2")); err == nil {
		actionID := first.String(ctx.Param("param3"), "view")

		return templateID, actionID, objectID
	}

	// Otherwise, we just parse action
	actionID := first.String(ctx.Param("param2"), "index")
	return templateID, actionID, primitive.NilObjectID
}

func buildAdmin_GetBuilder(ctx *steranko.Context, factory *service.Factory, session data.Session, template model.Template, actionID string, objectID primitive.ObjectID) (build.Builder, error) {

	const location = "handler.buildAdmin_GetBuilder"

	// Create the correct builder for this controller
	switch template.Model {

	case "Domain", "Search", "SSO", "Followers", "Following":
		return build.NewDomain(factory, session, ctx.Request(), ctx.Response(), template, actionID)

	case "Syndication":
		return build.NewSyndication(factory, session, ctx.Request(), ctx.Response(), template, actionID)

	case "Group":
		group := model.NewGroup()

		if !objectID.IsZero() {
			service := factory.Group()
			if err := service.LoadByID(session, objectID, &group); err != nil {
				return nil, derp.Wrap(err, location, "Unable to load Group", objectID)
			}
		}

		return build.NewGroup(factory, session, ctx.Request(), ctx.Response(), template, &group, actionID)

	case "Rule":

		rule := model.NewRule()

		if !objectID.IsZero() {
			authorization := getAuthorization(ctx)
			if err := factory.Rule().LoadByID(session, authorization.UserID, objectID, &rule); err != nil {
				return nil, derp.Wrap(err, location, "Unable to load Rule", objectID)
			}
		}

		return build.NewRule(factory, session, ctx.Request(), ctx.Response(), &rule, template, actionID)

	case "Stream":
		stream := model.NewStream()

		if !objectID.IsZero() {
			if err := factory.Stream().LoadByID(session, objectID, &stream); err != nil {
				return nil, derp.Wrap(err, location, "Unable to load Navigation stream", objectID)
			}
		}

		return build.NewNavigation(factory, session, ctx.Request(), ctx.Response(), template, &stream, actionID)

	case "Tag":
		searchTag := model.NewSearchTag()

		if !objectID.IsZero() {
			if err := factory.SearchTag().LoadByID(session, objectID, &searchTag); err != nil {
				return nil, derp.Wrap(err, location, "Unable to load Tag", searchTag)
			}
		}

		return build.NewSearchTag(factory, session, ctx.Request(), ctx.Response(), template, &searchTag, actionID)

	case "User":
		user := model.NewUser()

		if !objectID.IsZero() {
			if err := factory.User().LoadByID(session, objectID, &user); err != nil {
				return nil, derp.Wrap(err, location, "Unable to load User", objectID)
			}
		}

		return build.NewUser(factory, session, ctx.Request(), ctx.Response(), template, &user, actionID)

	case "Webhook":
		webhook := model.NewWebhook()

		if !objectID.IsZero() {
			if err := factory.Webhook().LoadByID(session, objectID, &webhook); err != nil {
				return nil, derp.Wrap(err, location, "Unable to load Webhook", objectID)
			}
		}

		return build.NewWebhook(factory, session, ctx.Request(), ctx.Response(), template, &webhook, actionID)

	default:
		return nil, derp.NotFound(location, "Template MODEL must be one of: 'Rule', 'Domain', 'Syndication', 'Group', 'Stream', 'Tag', or 'User'", template.Model)
	}
}
