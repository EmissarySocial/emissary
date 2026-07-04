package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// PostProxyURL is a handler for the "Proxy URL" endpoint, which allows clients to request that the server load a URL and return the result.
// This allows an actor's clients to\ access remote ActivityStreams objects which require authentication to access.
// This is also used by some clients to work around CORS issues when trying to load remote documents from the browser.
// https://www.w3.org/TR/activitypub/#proxyUrl
func PostProxyURL(context *steranko.Context, factory *service.Factory, session data.Session, user *model.User) error {

	const location = "handler.PostProxyURL"

	// Define the transaction we expect to receive.
	transaction := struct {
		URL string `form:"id" json:"id"`
	}{}

	// Bind the form data to the transaction object.
	if err := context.Bind(&transaction); err != nil {
		return derp.Wrap(err, location, "Unable to bind form data")
	}

	// RULE: Don't allow empty IDs
	if transaction.URL == "" {
		return derp.Validation("Parameter 'id' is required", transaction.URL)
	}

	// RULE: Remote value MUST be a valid URL
	if uri.NotValidURL(transaction.URL) {
		return derp.Validation("Parameter 'id' must be a valid URL", transaction.URL)
	}

	// Fast, friendly reject for obviously-local URLs. The authoritative SSRF
	// guard is the resolved-IP check in remote's transport.
	if uri.IsLocalURL(transaction.URL) {
		if uri.NotLocalHostname(factory.Hostname()) {
			return derp.Validation("Parameter 'id' must not be a local address", transaction.URL)
		}
	}

	// Create the ActivityStreams client
	activityService := factory.ActivityStream()
	client := activityService.UserClient(user.UserID)

	// Load the document from the URL (DON'T load from cache, but DO refresh the cache)
	result, err := client.Load(transaction.URL, ascache.WithWriteOnly())

	if err != nil {
		return derp.Wrap(err, location, "Unable to load URL", transaction.URL)
	}

	// Success
	return context.JSON(http.StatusOK, result.Value())
}
