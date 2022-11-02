package middleware

import (
	"github.com/benpate/html"
	"github.com/labstack/echo/v4"
)

// AllowCSR allows "same-site" authentication cookies to work on Cross-Site Requests for specific GET routes.
// It returns an empty HTML page that refreshes to the same URL.  Since this second request is now
// coming from the same site, cookies will be passed through on the second request.
func AllowCSR(next echo.HandlerFunc) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		if ctx.Request().Method == "GET" {

			// Get the current URL
			url := ctx.Request().URL
			query := url.Query()

			// If we don't already have the flag value
			if query.Get("__AllowCSR__") == "" {

				// Add the flag to the query string
				query.Set("__AllowCSR__", "true")
				url.RawQuery = query.Encode()
				urlString := url.String()

				// Build a simple HTML page that refreshes to the same URL
				b := html.New()

				b.Container("html")
				b.Container("head")
				b.Container("meta").Attr("http-equiv", "refresh").Attr("content", "0;url="+urlString)
				b.Close()
				b.Container("body")
				b.A(urlString).InnerHTML("Redirecting Now. Click here to continue...").Close()
				b.CloseAll()

				// Return success.
				return ctx.HTML(200, b.String())
			}
		}

		// Fall through means that we've already done the redirect,
		// so we SHOULD have the cookies now.
		return next(ctx)
	}
}
