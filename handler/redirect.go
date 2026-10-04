package handler

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

// RedirectTo returns a handler that sends every request to a fixed location
func RedirectTo(location string) func(ctx echo.Context) error {

	return func(ctx echo.Context) error {
		return ctx.Redirect(http.StatusSeeOther, location)
	}
}

// RedirectTag sends /tags/:tag -- the URL Mastodon clients use for a hashtag -- to
// that hashtag's search page.
func RedirectTag(ctx echo.Context) error {
	return ctx.Redirect(http.StatusSeeOther, "/search?q="+url.QueryEscape("#"+ctx.Param("tag")))
}
