package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/render"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// GetStream generates the base HTML for a stream
func GetStream(factoryManager *service.FactoryManager) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get the service factory
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Unrecognized domain"))
		}

		// Get the stream service
		templateService := factory.Template()
		streamService := factory.Stream()

		stream, err := streamService.LoadByToken(ctx.Param("token"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream"))
		}

		// Render inner content
		domainView := "page"

		if ctx.Request().Header.Get("hx-request") == "true" {
			domainView = "stream"
		}

		pipeline :=
			render.NewDomainWrapper(templateService,
				render.NewStreamWrapper(templateService, streamService, stream, ctx.QueryParam("view")),
				domainView,
			)

		result, err := pipeline.Render()

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering innerHTML"))
		}

		return ctx.HTML(http.StatusOK, result)
	}
}
