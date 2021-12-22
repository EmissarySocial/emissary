package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/domain"
	"github.com/benpate/ghost/first"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetAdmin handles GET requests
func GetAdmin(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetAdmin", "Unrecognized Domain")
		}

		sterankoContext := ctx.(*steranko.Context)
		renderer, err := getAdminRenderer(factory, sterankoContext)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetAdmin", "Error generating Renderer")
		}

		return renderPage(factory, sterankoContext, renderer)
	}
}

// PostAdmin handles POST/DELETE requests
func PostAdmin(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetAdmin", "Unrecognized Domain")
		}

		// Get Renderer
		sterankoContext := ctx.(*steranko.Context)
		renderer, err := getAdminRenderer(factory, sterankoContext)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetAdmin", "Error generating Renderer")
		}

		// Execute pipeline
		if action, ok := renderer.Action(); ok {
			if err := render.DoPipeline(factory, renderer, ctx.Response().Writer, action.Steps, render.ActionMethodPost); err != nil {
				return derp.Wrap(err, "ghost.handler.PostAdmin", "Error executing action pipeline", action)
			}
		}

		return nil
	}
}

// getAdminRenderer returns a fully initialized Renderer based on the request parameters
func getAdminRenderer(factory *domain.Factory, ctx *steranko.Context) (render.Renderer, error) {

	controller := first.String(ctx.Param("param1"), "domain")

	switch controller {

	case "analytics":
		action := first.String(ctx.Param("param2"), "index")
		result := render.NewDomain(factory, ctx, factory.Layout().Analytics(), action)
		return &result, nil

	case "domain":
		action := first.String(ctx.Param("param2"), "index")
		result := render.NewDomain(factory, ctx, factory.Layout().Domain(), action)
		return &result, nil

	case "toplevel":
		result, err := render.NewTopLevel(factory, ctx, factory.Layout().TopLevel(), ctx.Param("param2"), ctx.Param("param3"))
		return &result, err

	case "groups":
		group := model.NewGroup()

		if groupID, err := primitive.ObjectIDFromHex(ctx.Param("param2")); err == nil {
			groupService := factory.Group()

			if err := groupService.LoadByID(groupID, &group); err != nil {
				return nil, derp.Wrap(err, "ghost.handler.getAdminRenderer", "Error loading Group", groupID)
			}

			result := render.NewGroup(factory, ctx, &group, first.String(ctx.Param("param3"), "view"))
			return &result, nil
		}

		// Fall through means we aren't looking at a specific user
		result := render.NewGroup(factory, ctx, &group, first.String(ctx.Param("param2"), "index"))
		return &result, nil

	case "users":
		user := model.NewUser()

		if userID, err := primitive.ObjectIDFromHex(ctx.Param("param2")); err == nil {
			userService := factory.User()

			if err := userService.LoadByID(userID, &user); err != nil {
				return nil, derp.Wrap(err, "ghost.handler.getAdminRenderer", "Error loading User", userID)
			}

			result := render.NewUser(factory, ctx, &user, first.String(ctx.Param("param3"), "view"))
			return &result, nil
		}

		// Fall through means we aren't looking at a specific user
		result := render.NewUser(factory, ctx, &user, first.String(ctx.Param("param2"), "index"))
		return &result, nil

	default:
		return nil, derp.New(derp.CodeNotFoundError, "ghost.handler.getAdminRenderer", "Invalid Arguments", ctx.Param("controller"), ctx.Param("objectId"), ctx.Param("action"))
	}
}
