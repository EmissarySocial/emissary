package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetWebfinger returns public webfinger information for a designated user.
// https://webfinger.net
// WebFinger data based on https://docs.joinmastodon.org/spec/webfinger/
func GetWebfinger(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.GetWebfinger"

	// Use the Locator service to find the WebFinger resource
	resourceID := ctx.QueryParam("resource")
	resource, err := factory.Locator().GetWebFingerResult(resourceID)

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving WebFinger resource", derp.WithCode(http.StatusBadRequest))
	}

	// If relation is specified, then limit links to that type only
	resource.FilterLinks(ctx.QueryParam("rel"))
	ctx.Response().Header().Set("Content-Type", model.MimeTypeJSONResourceDescriptorWithCharset)

	// Return the Response as a JSON object
	return ctx.JSON(http.StatusOK, resource)
}
