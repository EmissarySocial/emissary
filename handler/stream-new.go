package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// GetNewStream generates an HTML form where authenticated users can create a new stream
func GetNewStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Locate the domain we're working in
		factory, err := factoryManager.ByContext(ctx)
		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error locating domain"))
		}

		streamService := factory.Stream()
		stream, err := streamService.NewWithTemplate(ctx.Param("stream"), ctx.Param("template"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error creating new stream"))
		}

		// Render the HTML
		// Render page content (full or partial)
		renderer := factory.FormRenderer(*stream, "stream-new.html", "create")
		result, err := renderPage(factory.Layout(), renderer, isFullPageRequest(ctx))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering HTML"))
		}

		return ctx.HTML(http.StatusOK, result)
	}
}

func PostNewStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get data from Form POST
		data := make(map[string]interface{})

		if err := ctx.Bind(data); err != nil {
			return derp.Wrap(err, "ghost.handler.PostNewStream", "Can't bind POST data")
		}

		// Locate the domain we're working in
		factory, err := factoryManager.ByContext(ctx)
		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostNewStream", "Error locating domain"))
		}

		// Get the steam service and the new stream
		streamService := factory.Stream()
		stream, err := streamService.NewWithTemplate(ctx.Param("stream"), ctx.Param("template"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostNewStream", "Error creating new stream"))
		}

		// Execute "create" transition
		transition, err := streamService.Transition(stream, "create", data);
		
		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostNewStream", "Error performing transition"))
		}

		// Render result
		html, err := factory.StreamRenderer(*stream, transition.NextView).Render()

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.PostNewStream", "Error rendering next view"))
		}

		return ctx.HTML(200, html)
	}
}
