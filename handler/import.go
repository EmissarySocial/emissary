package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetImportedURL maps remote URLs for imported items to local URLs
// This is the "Oracle" in the W3C Data Portability standard:
// https://swicg.github.io/activitypub-data-portability/lola
func GetImportedURL(ctx *steranko.Context, factory *service.Factory, session data.Session) error {
	const location = "handler.GetImportedURL"

	// Collect the URL parameter we're going to map
	originalURL := ctx.QueryParam("url")

	if originalURL == "" {
		return derp.NotFound(location, "URL parameter must be provided")
	}

	// Try to load the ImportItem that maps to this URL
	importItemService := factory.ImportItem()
	importItem := model.NewImportItem()

	if err := importItemService.LoadByRemoteURL(session, originalURL, &importItem); err != nil {
		return derp.Wrap(err, location, "Unable to load ImportItem by URL", "url: "+originalURL)
	}

	// 404 if there's no local mapping
	if importItem.LocalURL == "" {
		return derp.NotFound(location, "No local URL found for that remote URL", "url: "+originalURL)
	}

	// Redirect the request to the correct local URL.
	return ctx.Redirect(http.StatusPermanentRedirect, importItem.LocalURL)
}
