package handler

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

// GetWebfinger returns public webfinger information for a designated user.
// WebFinger data based on https://docs.joinmastodon.org/spec/webfinger/
func GetWebfinger(fm *server.Factory) echo.HandlerFunc {

	location := "handler.GetWebFinger"

	return func(ctx echo.Context) error {

		var resource digit.Resource
		var err error

		// Validate Domain and Get Factory Object
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		resourceID := ctx.QueryParam("resource")

		// Handle User Requests
		if strings.HasPrefix(resourceID, "acct:") {
			userService := factory.User()
			username := strings.TrimPrefix(resourceID, "acct:")
			resource, err = userService.LoadWebFinger(username)
		} else {
			streamService := factory.User()
			resource, err = streamService.LoadWebFinger(resourceID)
		}

		// Handle Errors
		if err != nil {
			return derp.NewBadRequestError(location, "Invalid Resource", resourceID)
		}

		spew.Dump(resource)

		// If relation is specified, then limit links to that type only
		resource.FilterLinks(ctx.QueryParam("rel"))

		// Return the Response as a JSON object
		return ctx.JSON(http.StatusOK, resource)
	}
}
