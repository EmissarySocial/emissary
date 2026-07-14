package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// fullURL returns the URL for a request that include the protocol, hostname, and path
func fullURL(factory *service.Factory, ctx echo.Context) string {
	return factory.Host() + ctx.Request().URL.String()
}

// getAuthorization extracts a model.Authorization record from the steranko.Context
func getAuthorization(ctx *steranko.Context) model.Authorization {

	if claims, err := ctx.Authorization(); err == nil {

		if auth, ok := claims.(*model.Authorization); ok {
			return *auth
		}
	}

	return model.NewAuthorization()
}
