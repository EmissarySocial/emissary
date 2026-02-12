package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/vocab"
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

	// Set header and serve JSON-LD document
	context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)

	if err := context.JSON(http.StatusOK, object.GetJSONLD()); err != nil {
		return true, derp.Wrap(err, location, "Unable to generate JSON-LD", object)
	}

	return true, nil
}
