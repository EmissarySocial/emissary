package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/labstack/echo/v4"
)

// handleJSONLD determines if the client has requested a document encoded as ActivityPub/JSON-LD/JSON.
// If so, it returns TRUE, and writes the JSON-LD document to the response (and an improbable error).
// If the client has NOT requested a JSON-LD document, then it returns FALSE, and no error.
func handleJSONLD(context echo.Context, object model.JSONLDGetter) (bool, error) {

	const location = "handler.handleJSONLD"

	// Ignore non-activitypub requests
	if hannibal.NotActivityPubRequest(context.Request()) {
		return false, nil
	}

	// Set headers and serve JSON-LD document
	headers.SetVariant(context.Response().Header(), headers.VariantActivityPub)

	if err := context.JSON(http.StatusOK, object.GetJSONLD()); err != nil {
		return true, derp.Wrap(err, location, "Generating JSON-LD", object)
	}

	return true, nil
}
