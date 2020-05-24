package handler

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
	"github.com/benpate/presto"
	"github.com/labstack/echo/v4"
)

// GetStream generates the base HTML for a stream
func GetStream(maker service.FactoryMaker, roles ...presto.RoleFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Get the service factory
		factory := maker.Factory(ctx.Request().Context())

		// Get the stream service
		streamService := factory.Stream()

		scopes := presto.ScopeFuncSlice{}
		roles := presto.RoleFuncSlice{}

		// Try to load the stream from the database (with all presto decorations)
		code, object := presto.Get(ctx, streamService, nil, scopes, roles)

		// ERROR..  SHOULD PROBABLY HAVE A BETTER ERROR PAGE HERE...
		if object == nil {
			return ctx.String(code, "")
		}

		stream, ok := object.(*model.Stream)

		if ok == false {
			derp.New(500, "handler.GetStream", "Unrecognized variable returned by Stream service", object).Report()
			return ctx.String(500, "")
		}

		// Use the service.Template to manage HTML templates
		templateCache := factory.TemplateCache()

		// Generate the result
		result, err := templateCache.Render(stream.Data)

		if err != nil {
			return ctx.String(err.Code, "")
		}

		// Return to caller
		return ctx.HTML(http.StatusOK, result)
	}
}
