package handler

import (
	"net/http"

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
		code, stream := presto.Get(ctx, streamService, nil, scopes, roles)

		// ERROR..  SHOULD PROBABLY HAVE A BETTER ERROR PAGE HERE...
		if stream == nil {
			return ctx.String(code, "")
		}

		// Use the service.Template to manage HTML templates
		templateService := factory.Template()

		// Generate the result
		result := templateService.HTML(stream)

		// Return to caller
		return ctx.HTML(http.StatusOK, result)
	}
}
