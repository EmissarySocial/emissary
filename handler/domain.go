package handler

import (
	"github.com/benpate/derp"
	"github.com/benpate/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetAdmin handles GET requests
func GetAdmin(factoryManager *server.Factory) echo.HandlerFunc {
	return adminRenderer(factoryManager, render.ActionMethodGet)
}

// PostAdmin handles POST/DELETE requests
func PostAdmin(factoryManager *server.Factory) echo.HandlerFunc {
	return adminRenderer(factoryManager, render.ActionMethodPost)
}

func adminRenderer(factoryManager *server.Factory, actionMethod render.ActionMethod) echo.HandlerFunc {

	const location = "handler.adminRenderer"

	return func(ctx echo.Context) error {

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Authenticate the page request
		sterankoContext := ctx.(*steranko.Context)

		// Only domain owners can access admin pages
		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewForbiddenError("whisper.handler.adminRenderer", "Unauthorized")
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
			renderer, err = render.NewDomain(factory, sterankoContext, layout, action)

			if err != nil {
				return derp.Wrap(err, "whisper.handler.adminRenderer", "Error loading domain")
			}

		case "domain":
			layout := layoutService.Domain()
			action := layout.Action(actionID)
			renderer, err = render.NewDomain(factory, sterankoContext, layout, action)

			if err != nil {
				return derp.Wrap(err, "whisper.handler.adminRenderer", "Error loading domain")
			}

		case "groups":
			group := model.NewGroup()

			if !objectID.IsZero() {
				service := factory.Group()
				if err := service.LoadByID(objectID, &group); err != nil {
					return derp.Wrap(err, "whisper.handler.adminRenderer", "Error loading Group", objectID)
				}
			}

			renderer, err = render.NewRenderer(factory, sterankoContext, &group, actionID)

		case "toplevel":
			stream := model.NewStream()

			if !objectID.IsZero() {
				service := factory.Stream()
				if err := service.LoadByID(objectID, &stream); err != nil {
					return derp.Wrap(err, "whisper.handler.adminRenderer", "Error loading TopLevel stream", objectID)
				}
			}

			renderer, err = render.NewRenderer(factory, sterankoContext, &stream, actionID)

		case "users":
			user := model.NewUser()

			if !objectID.IsZero() {
				service := factory.User()
				if err := service.LoadByID(objectID, &user); err != nil {
					return derp.Wrap(err, "whisper.handler.adminRenderer", "Error loading User", objectID)
				}
			}

			renderer, err = render.NewRenderer(factory, sterankoContext, &user, actionID)

		default:
			return derp.NewNotFoundError("whisper.handler.getAdminRenderer", "Invalid Arguments", ctx.Param("param1"), ctx.Param("param2"), ctx.Param("param3"))
		}

		// If this is a POST, then execute the action pipeline
		if actionMethod == render.ActionMethodPost {

			if err := render.DoPipeline(renderer, ctx.Response().Writer, renderer.Action().Steps, actionMethod); err != nil {
				return derp.Wrap(err, "whisper.handler.PostAdmin", "Error executing action pipeline", renderer.Action())
			}
			return nil
		}

		// Otherwise, use the standard "renderPage" function to return HTML
		return renderPage(factory, sterankoContext, renderer)
	}
}
