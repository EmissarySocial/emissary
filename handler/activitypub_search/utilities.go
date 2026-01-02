package activitypub_search

import (
	"github.com/EmissarySocial/emissary/service"
	"github.com/labstack/echo/v4"
)

// fullURL returns the URL for a request that include the protocol, hostname, and path
func fullURL(factory *service.Factory, ctx echo.Context) string {
	return factory.Host() + ctx.Request().URL.String()
}
