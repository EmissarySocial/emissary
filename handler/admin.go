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
	return renderAdmin(factoryManager, render.ActionMethodGet)
}

// PostAdmin handles POST/DELETE requests
func PostAdmin(factoryManager *server.Factory) echo.HandlerFunc {
	return renderAdmin(factoryManager, render.ActionMethodPost)
}

func renderAdmin(factoryManager *server.Factory, actionMethod render.ActionMethod) echo.HandlerFunc {

	const location = "handler.adminRenderer"

	return func(ctx echo.Context) error {

		// Authenticate the page request
		sterankoContext := ctx.(*steranko.Context)

		// Only domain owners can access admin pages
		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewForbiddenError(location, "Unauthorized")
		}

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		var renderer render.Renderer
		var objectID primitive.ObjectID
		var actionID string

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
			layout := factory.Layout().Analytics()
			service := factory.Domain()
			object := model.NewDomain()
			if err := service.Load(&object); err != nil {
				return derp.Wrap(err, location, "Error loading Group", objectID)
			}
			renderer, err = render.NewDomain(factory, sterankoContext, layout, &object, actionID)

		case "appearance":
			layout := factory.Layout().Appearance()
			service := factory.Domain()
			object := model.NewDomain()
			if err := service.Load(&object); err != nil {
				return derp.Wrap(err, location, "Error loading Group", objectID)
			}
			renderer, err = render.NewDomain(factory, sterankoContext, layout, &object, actionID)

		case "connections":
			layout := factory.Layout().Connections()
			service := factory.Domain()
			object := model.NewDomain()
			if err := service.Load(&object); err != nil {
				return derp.Wrap(err, location, "Error loading Group", objectID)
			}
			renderer, err = render.NewDomain(factory, sterankoContext, layout, &object, actionID)

		case "domain":
			layout := factory.Layout().Domain()
			service := factory.Domain()
			object := model.NewDomain()
			if err := service.Load(&object); err != nil {
				return derp.Wrap(err, location, "Error loading Group", objectID)
			}
			renderer, err = render.NewDomain(factory, sterankoContext, layout, &object, actionID)

		case "groups":
			group := model.NewGroup()

			if !objectID.IsZero() {
				service := factory.Group()
				if err := service.LoadByID(objectID, &group); err != nil {
					return derp.Wrap(err, location, "Error loading Group", objectID)
				}
			}

			renderer, err = render.NewGroup(factory, sterankoContext, &group, actionID)

		case "toplevel":
			stream := model.NewStream()

			if !objectID.IsZero() {
				service := factory.Stream()
				if err := service.LoadByID(objectID, &stream); err != nil {
					return derp.Wrap(err, location, "Error loading TopLevel stream", objectID)
				}
			}

			renderer, err = render.NewTopLevel(factory, sterankoContext, &stream, actionID)

		case "users":
			user := model.NewUser()

			if !objectID.IsZero() {
				service := factory.User()
				if err := service.LoadByID(objectID, &user); err != nil {
					return derp.Wrap(err, location, "Error loading User", objectID)
				}
			}

			renderer, err = render.NewUser(factory, sterankoContext, &user, actionID)

		default:
			return derp.NewNotFoundError(location, "Invalid Arguments", ctx.Param("param1"), ctx.Param("param2"), ctx.Param("param3"))
		}

		// Error handler for all renderers
		if err != nil {
			return derp.Wrap(err, location, "Error generating renderer")
		}

		// Success!!
		return renderPage(factory, sterankoContext, renderer, actionMethod)
	}
}
