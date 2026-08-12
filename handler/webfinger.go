package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetWebfinger returns public webfinger information for a designated user.
// https://webfinger.net
// WebFinger data based on https://docs.joinmastodon.org/spec/webfinger/
func GetWebfinger(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetWebfinger"

	resourceID := ctx.QueryParam("resource")

	// RULE: RFC 7033 Section 4.2 requires a 400 when the `resource` parameter is missing. Falling
	// through would look the empty string up as a local username, and a bare
	// `GET /.well-known/webfinger` is a common probe, so this is reached by accident and not only
	// by construction.
	if resourceID == "" {
		return derp.BadRequest(location, "Missing required parameter: resource")
	}

	// Use the Locator service to find the WebFinger resource
	resource, err := factory.Locator().GetWebFingerResult(session, resourceID)

	// The Locator already distinguishes "malformed" (400) from "we do not have that" (404), so the
	// error code is passed through rather than flattened -- a resource on another host, or a User
	// who is hidden from public discovery, is a 404 and must not be reported as a bad request.
	if err != nil {
		return derp.Wrap(err, location, "Retrieving WebFinger resource")
	}

	// If relation is specified, then limit links to that type only
	resource.FilterLinks(ctx.QueryParam("rel"))
	ctx.Response().Header().Set("Content-Type", model.MimeTypeJSONResourceDescriptorWithCharset)

	// Return the Response as a JSON object
	return ctx.JSON(http.StatusOK, resource)
}
