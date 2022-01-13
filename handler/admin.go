package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/first"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetAdmin handles GET requests
func GetAdmin(factoryManager *server.FactoryManager) echo.HandlerFunc {
	return adminRenderer(factoryManager, render.ActionMethodGet)
}

// PostAdmin handles POST/DELETE requests
func PostAdmin(factoryManager *server.FactoryManager) echo.HandlerFunc {
	return adminRenderer(factoryManager, render.ActionMethodPost)
}

func adminRenderer(factoryManager *server.FactoryManager, actionMethod render.ActionMethod) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetAdmin", "Unrecognized Domain")
		}

		// Authenticate the page request
		sterankoContext := ctx.(*steranko.Context)

		// Only domain owners can access admin pages
		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewForbiddenError("ghost.handler.adminRenderer", "Unauthorized")
		}

		var renderer render.Renderer
		var objectID primitive.ObjectID
		var actionID string

		layoutService := factory.Layout()

		// Parse request arguments
		controller := first.String(ctx.Param("param1"), "domain")

		if id, err := primitive.ObjectIDFromHex(ctx.Param("param2")); err == nil {
			actionID = first.String(ctx.Param("param3"), "view")
			objectID = id
		} else {
			actionID = first.String(ctx.Param("param2"), "index")
			objectID = primitive.NilObjectID
		}

		// Create the correct renderer for this controller
		switch controller {

		case "analytics":
			layout := layoutService.Analytics()
			action := layout.Action(actionID)
			renderer = render.NewDomain(factory, sterankoContext, layout, action)

		case "domain":
			layout := layoutService.Domain()
			action := layout.Action(actionID)
			renderer = render.NewDomain(factory, sterankoContext, layout, action)

		case "toplevel":
			layout := layoutService.TopLevel()
			action := layout.Action(actionID)
			service := factory.Stream()
			stream := model.NewStream()

			if !objectID.IsZero() {
				if err := service.LoadByID(objectID, &stream); err != nil {
					return derp.Wrap(err, "ghost.handler.adminRenderer", "Error loading TopLevel stream", objectID)
				}
			}

			renderer = render.NewTopLevel(factory, sterankoContext, layout, action, &stream)

		case "groups":
			layout := layoutService.Group()
			action := layout.Action(actionID)
			service := factory.Group()
			group := model.NewGroup()

			if !objectID.IsZero() {
				if err := service.LoadByID(objectID, &group); err != nil {
					return derp.Wrap(err, "ghost.handler.adminRenderer", "Error loading Group", objectID)
				}
			}

			renderer = render.NewGroup(factory, sterankoContext, layout, action, &group)

		case "users":
			layout := layoutService.User()
			action := layout.Action(actionID)
			service := factory.User()
			user := model.NewUser()

			if !objectID.IsZero() {
				if err := service.LoadByID(objectID, &user); err != nil {
					return derp.Wrap(err, "ghost.handler.adminRenderer", "Error loading User", objectID)
				}
			}

			renderer = render.NewUser(factory, sterankoContext, layout, action, &user)

		default:
			return derp.NewNotFoundError("ghost.handler.getAdminRenderer", "Invalid Arguments", ctx.Param("param1"), ctx.Param("param2"), ctx.Param("param3"))
		}

		// If this is a POST, then execute the action pipeline
		if actionMethod == render.ActionMethodPost {

			if err := render.DoPipeline(renderer, ctx.Response().Writer, renderer.Action().Steps, actionMethod); err != nil {
				return derp.Wrap(err, "ghost.handler.PostAdmin", "Error executing action pipeline", renderer.Action())
			}
			return nil
		}

		// Otherwise, use the standard "renderPage" function to return HTML
		return renderPage(factory, sterankoContext, renderer)
	}
}
