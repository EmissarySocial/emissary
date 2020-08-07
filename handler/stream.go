package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/benpate/presto"
	"github.com/benpate/presto/scope"
	"github.com/labstack/echo/v4"
)

// GetStream generates the base HTML for a stream
func GetStream(maker service.FactoryMaker, roles ...presto.RoleFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get the service factory
		factory := maker.Factory(ctx.Request().Context())
		defer factory.Close()

		// Get the stream service
		streamService := factory.Stream()

		scopes := scope.And(scope.String("token"), scope.NotDeleted)

		criteria, err := scopes(ctx)

		if err != nil {
			err = derp.Wrap(err, "ghost.handler.GetStream", "Error evaluating scope")
			derp.Report(err)
			return err
		}

		stream, err := streamService.Load(criteria)

		if err != nil {
			err = derp.Wrap(err, "ghost.handler.GetStream", "Error loading stream from service", criteria)
			derp.Report(err)
			return err
		}

		pageService := factory.PageService()

		var header string
		var footer string

		if ctx.Request().Header.Get("HX-Request") == "" {
			header, footer = pageService.RenderPage(stream, ctx.Param("view"))
		} else {
			header, footer = pageService.RenderPartial(stream, ctx.Param("view"))
		}

		// Generate the result
		result, err := streamService.Render(stream, ctx.Param("view"))

		if err != nil {
			derp.Report(err)
			return ctx.String(err.Code, "")
		}

		// Return to caller
		return ctx.HTML(http.StatusOK, header+result+footer)
	}
}
