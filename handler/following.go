package handler

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

// GetFollowingTunnel redirects legacy remote-follow links to the "following-edit" settings page.
// Published URLs (WebFinger, outbox templates) now point directly at /@me/settings/following-edit,
// but remote servers cache WebFinger responses, so old tunnel links must keep working.
func GetFollowingTunnel(ctx echo.Context) error {
	target := "/@me/settings/following-edit?url=" + url.QueryEscape(ctx.QueryParam("uri"))
	return ctx.Redirect(http.StatusFound, target)
}
