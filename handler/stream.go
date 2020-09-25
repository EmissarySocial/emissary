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
		streamService := factory.Stream()
		stream, err := streamService.LoadByToken(ctx.Param("token"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream from service"))
		}

		// Render inner content
		streamWrapper := render.NewStreamWrapper(factory, stream)
		innerHTML, err := streamWrapper.Render(ctx.Param("view"))

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering innerHTML"))
		}

		// Render wrapper content
		domainWrapper := render.NewDomainWrapper(factory, streamWrapper, innerHTML)
		domainView := getDomainView(ctx.Request())
		result, err := domainWrapper.Render(domainView)

		if err != nil {
			return derp.Report(derp.Wrap(err, "ghost.handler.GetStream", "Error rendering wrapper"))
		}

		return ctx.HTML(http.StatusOK, *result)
	}
}

func getDomainView(r *http.Request) string {

	if r.Header.Get("hx-request") == "true" {
		return "stream"
	}

	return "page"
}
