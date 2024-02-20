package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/labstack/echo/v4"
)

// GetWebfinger returns public webfinger information for a designated user.
// WebFinger data based on https://docs.joinmastodon.org/spec/webfinger/
func GetWebfinger(fm *server.Factory) echo.HandlerFunc {

	location := "handler.GetWebFinger"

	return func(ctx echo.Context) error {

		// Validate Domain and Get Factory Object
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		resourceID := ctx.QueryParam("resource")

		// Look for Users first
		if resource, err := factory.User().LoadWebFinger(resourceID); err == nil {
			return writeResource(ctx, resource)
		}

		// Next, look for Streams (that are ActivityPub actors)
		if resource, err := factory.Stream().LoadWebFinger(resourceID); err == nil {
			return writeResource(ctx, resource)
		}

		// Otherwise, break unceremoniously
		return derp.NewBadRequestError(location, "Invalid Resource", resourceID)
	}
}

func writeResource(ctx echo.Context, resource digit.Resource) error {

	// If relation is specified, then limit links to that type only
	resource.FilterLinks(ctx.QueryParam("rel"))
	ctx.Response().Header().Set("Content-Type", model.MimeTypeJSONResourceDescriptor)

	// Return the Response as a JSON object
	return ctx.JSON(http.StatusOK, resource)
}
