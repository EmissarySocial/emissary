package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/labstack/echo/v4"
	"github.com/timewasted/go-accept-headers"
)

func isJSONLDRequest(context echo.Context) bool {

	acceptHeader := context.Request().Header.Get("Accept")
	contentType, _ := accept.Negotiate(acceptHeader, vocab.ContentTypeHTML, vocab.ContentTypeActivityPub, vocab.ContentTypeJSONLD, vocab.ContentTypeJSON)

	switch contentType {
	case "application/activity+json", "application/ld+json", "application/json", "text/json":
		return true
	default:
		return false
	}
}

// handleJSONLD determines if the client has requested a document encoded as ActivityPub/JSON-LD/JSON.
// If so, it returns TRUE, and writes the JSON-LD document to the response (and an improbable error).
// If the client has NOT requested a JSON-LD document, then it returns FALSE, and no error.
func handleJSONLD(context echo.Context, object model.JSONLDGetter) (bool, error) {

	if isJSONLDRequest(context) {
		context.Response().Header().Set(vocab.ContentType, vocab.ContentTypeActivityPub)
		return true, context.JSON(http.StatusOK, object.GetJSONLD())
	}

	return false, nil
}
