package handler

import (
	"bytes"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/server"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetDomain handles GET requests
func GetDomain(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.GetDomain", "Unrecognized Domain")
		}

		// Try to load the Domain from the database
		domainService := factory.Domain()
		domain := model.NewDomain()

		if err := domainService.Load(&domain); err != nil {
			return derp.Wrap(err, "ghost.handler.GetDomain", "Error loading Domain object")
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		renderer := render.NewDomain(factory, sterankoContext, domain, ctx.Param("action"))

		// Partial Page requests are simpler.
		if renderer.IsPartialRequest() {
			result, err := renderer.Render()

			if err != nil {
				return derp.Wrap(err, "ghost.handler.GetDomain", "Error rendering domain")
			}
			return ctx.HTML(http.StatusOK, string(result))
		}

		// Full Page requests require the layout service
		layoutService := factory.Layout()
		template := layoutService.Global().HTMLTemplate
		var buffer bytes.Buffer

		if err := template.ExecuteTemplate(&buffer, "page", &renderer); err != nil {
			return derp.Wrap(err, "ghost.handler.GetDomain", "Error rendering full-page content")
		}

		return ctx.HTML(http.StatusOK, buffer.String())
	}
}

// PostDomain handles POST/DELETE requests
func PostDomain(factoryManager *server.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "ghost.handler.PostDomain", "Unrecognized Domain")
		}

		// Try to load the Domain from the database
		domainService := factory.Domain()
		domain := model.NewDomain()

		if err := domainService.Load(&domain); err != nil {
			return derp.Wrap(err, "ghost.handler.PostDomain", "Error loading Domain object")
		}

		// Try to find the action requested by the user.  This also enforces user permissions...
		sterankoContext := ctx.(*steranko.Context)
		renderer := render.NewDomain(factory, sterankoContext, domain, ctx.Param("action"))

		if action, ok := renderer.Action(); ok {
			if err := render.DoPipeline(factory, renderer, ctx.Response().Writer, action.Steps, render.ActionMethodPost); err != nil {
				return derp.Wrap(err, "ghost.handler.PostDomain", "Error executing Action", action)
			}
		}

		return nil
	}
}
