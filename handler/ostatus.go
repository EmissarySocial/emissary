package handler

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/digit"
	"github.com/labstack/echo/v4"
)

// PostOStatusDiscover looks up a user's profile using oStatus discovery.
// If successful, it returns a redirect to the user's follow-request page.
// If unsuccessful, it returns a 200 with an English-language error message.
func PostOStatusDiscover(serverFactory *server.Factory) echo.HandlerFunc {

	return func(context echo.Context) error {

		var transaction struct {
			LocalAccount  string `form:"localAccount"`
			RemoteAccount string `form:"remoteAccount"`
		}

		// Get the transaction data from the form
		if err := context.Bind(&transaction); err != nil {
			return err
		}

		// Use WebFinger to get information about the user's account.
		resource, err := digit.Lookup(transaction.RemoteAccount)

		if err != nil {
			return context.String(http.StatusOK, "Your account doesn't support auto-discovery.")
		}

		// Try to find the subscribe request link
		link := resource.FindLink(digit.RelationTypeSubscribeRequest)

		if link.IsEmpty() {
			return context.String(http.StatusOK, "Your account doesn't support autodiscovery. (ERROR:B)")
		}

		// Replace the {uri} placeholder with the actual LOCAL account name
		forwardToURL := strings.ReplaceAll(link.Href, "{uri}", transaction.LocalAccount)

		// HTMX Redirect to the subscribe request page.
		context.Response().Header().Set("HX-Redirect", forwardToURL)
		return context.String(http.StatusOK, "")
	}
}
