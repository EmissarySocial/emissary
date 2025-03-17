package handler

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetWebfinger returns public webfinger information for a designated user.
// https://webfinger.net
// WebFinger data based on https://docs.joinmastodon.org/spec/webfinger/
func GetWebfinger(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.GetWebFinger"

	resourceID := ctx.QueryParam("resource")

	if !strings.HasPrefix(resourceID, "acct:") {
		return derp.NewBadRequestError(location, "Unrecognized Resource", resourceID)
	}

	resourceID = strings.TrimPrefix(resourceID, "acct:")

	// First, try the service account on the domain
	if resource, err := factory.Domain().LoadWebFinger(resourceID); err == nil {
		return writeResource(ctx, resource)
	}

	// Next, look for search actors (it's an easy check)
	if resource, err := factory.SearchQuery().LoadWebFinger(resourceID); err == nil {
		return writeResource(ctx, resource)
	}

	// Next, look for Users
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

func writeResource(ctx echo.Context, resource digit.Resource) error {

	// If relation is specified, then limit links to that type only
	resource.FilterLinks(ctx.QueryParam("rel"))
	ctx.Response().Header().Set("Content-Type", model.MimeTypeJSONResourceDescriptorWithCharset)

	// Return the Response as a JSON object
	return ctx.JSON(http.StatusOK, resource)
}
