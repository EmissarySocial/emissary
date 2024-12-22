package handler

import (
	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetAdmin handles GET requests
func GetAdmin(factoryManager *server.Factory) echo.HandlerFunc {
	return buildAdmin(factoryManager, build.ActionMethodGet)
}

// PostAdmin handles POST/DELETE requests
func PostAdmin(factoryManager *server.Factory) echo.HandlerFunc {
	return buildAdmin(factoryManager, build.ActionMethodPost)
}

func buildAdmin(factoryManager *server.Factory, actionMethod build.ActionMethod) echo.HandlerFunc {

	const location = "handler.adminBuilder"

	return func(ctx echo.Context) error {

		// Authenticate the page request
		sterankoContext := ctx.(*steranko.Context)

		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewForbiddenError(location, "Unauthorized")
		}

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
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
		builder, err := buildAdmin_GetBuilder(factory, sterankoContext, template, actionID, objectID)

		if err != nil {
			return derp.Wrap(err, location, "Error generating builder")
		}

		// Success!!
		return build.AsHTML(factory, sterankoContext, builder, actionMethod)
	}
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

func buildAdmin_GetBuilder(factory *domain.Factory, ctx *steranko.Context, template model.Template, actionID string, objectID primitive.ObjectID) (build.Builder, error) {

	const location = "handler.buildAdmin_GetBuilder"

	// Create the correct builder for this controller
	switch template.Model {

	case "domain":
		return build.NewDomain(factory, ctx.Request(), ctx.Response(), template, actionID)

	case "syndication":
		return build.NewSyndication(factory, ctx.Request(), ctx.Response(), template, actionID)

	case "group":
		group := model.NewGroup()

		if !objectID.IsZero() {
			service := factory.Group()
			if err := service.LoadByID(objectID, &group); err != nil {
				return nil, derp.Wrap(err, location, "Error loading Group", objectID)
			}
		}

		return build.NewGroup(factory, ctx.Request(), ctx.Response(), template, &group, actionID)

	case "rule":

		ruleService := factory.Rule()
		rule := model.NewRule()

		if !objectID.IsZero() {
			authorization := getAuthorization(ctx)
			if err := ruleService.LoadByID(authorization.UserID, objectID, &rule); err != nil {
				return nil, derp.Wrap(err, location, "Error loading Rule", objectID)
			}
		}

		return build.NewRule(factory, ctx.Request(), ctx.Response(), &rule, template, actionID)

	case "stream":
		stream := model.NewStream()

		if !objectID.IsZero() {
			service := factory.Stream()
			if err := service.LoadByID(objectID, &stream); err != nil {
				return nil, derp.Wrap(err, location, "Error loading Navigation stream", objectID)
			}
		}

		return build.NewNavigation(factory, ctx.Request(), ctx.Response(), template, &stream, actionID)

	case "tag":
		searchTag := model.NewSearchTag()

		if !objectID.IsZero() {
			service := factory.SearchTag()
			if err := service.LoadByID(objectID, &searchTag); err != nil {
				return nil, derp.Wrap(err, location, "Error loading Tag", searchTag)
			}
		}

		return build.NewSearchTag(factory, ctx.Request(), ctx.Response(), template, &searchTag, actionID)

	case "user":
		user := model.NewUser()

		if !objectID.IsZero() {
			service := factory.User()
			if err := service.LoadByID(objectID, &user); err != nil {
				return nil, derp.Wrap(err, location, "Error loading User", objectID)
			}
		}

		return build.NewUser(factory, ctx.Request(), ctx.Response(), template, &user, actionID)

	case "webhook":
		webhook := model.NewWebhook()

		if !objectID.IsZero() {
			service := factory.Webhook()
			if err := service.LoadByID(objectID, &webhook); err != nil {
				return nil, derp.Wrap(err, location, "Error loading User", objectID)
			}
		}

		return build.NewWebhook(factory, ctx.Request(), ctx.Response(), template, &webhook, actionID)

	default:
		return nil, derp.NewNotFoundError(location, "Template MODEL must be one of: 'rule', 'domain', 'syndication', 'group', 'stream', or 'user'", template.Model)
	}
}
