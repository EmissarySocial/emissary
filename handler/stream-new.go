package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)


func GetNewStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Locate the domain we're working in
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error locating domain"))
		}

		// Load the parent stream to validate permissions
		streamService := factory.Stream()
		parent, err := streamService.LoadByToken(ctx.Param("stream"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error loading parent stream"))
		}

		// Load the requested template
		templateService := factory.Template()
		template, err := templateService.Load(ctx.Param("template"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error loading Template"))
		}

		// Verify that the child stream can be placed inside the parent
		if !template.CanBeContainedBy(parent.Template) {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Invalid template"))
		}

		// Create a new stream
		stream := streamService.New()
		stream.Template = template.TemplateID

		// Render the HTML
		// Render page content (full or partial)
		renderer := factory.FormRenderer(*stream, "create")
		result, err := renderPage(factory.Layout(), renderer, isFullPageRequest(ctx))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering HTML"))
		}

		return ctx.HTML(http.StatusOK, result)

		return nil
	}
}

func PostNewStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		
		/*
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetNewStream", "Error locating domain"))
		}
		*/
		return nil
	}
}
