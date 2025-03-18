package activitypub_search

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/labstack/echo/v4"
)

// fullURL returns the URL for a request that include the protocol, hostname, and path
func fullURL(factory *domain.Factory, ctx echo.Context) string {
	return factory.Host() + ctx.Request().URL.String()
}
