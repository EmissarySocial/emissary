package handler

import (
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/digit"
	"github.com/benpate/steranko"
)

// PostOStatusDiscover looks up a user's profile using oStatus discovery.
// If successful, it returns a redirect to the user's follow-request page.
// If unsuccessful, it returns a 200 with an English-language error message.
func PostOStatusDiscover(ctx *steranko.Context, factory *domain.Factory) error {

	var transaction struct {
		LocalAccount  string `form:"localAccount"`
		RemoteAccount string `form:"remoteAccount"`
	}

	// Get the transaction data from the form
	if err := ctx.Bind(&transaction); err != nil {
		return err
	}

	// Use WebFinger to get information about the user's account.
	resource, err := digit.Lookup(transaction.RemoteAccount)

	if err != nil {
		return ctx.String(http.StatusOK, "Your account doesn't support auto-discovery.")
	}

	// Try to find the subscribe request link
	link := resource.FindLink(digit.RelationTypeSubscribeRequest)

	if link.IsEmpty() {
		return ctx.String(http.StatusOK, "Your account doesn't support autodiscovery. (ERROR:B)")
	}

	// Replace the {uri} placeholder with the actual LOCAL account name
	forwardToURL := strings.ReplaceAll(link.Template, "{uri}", transaction.LocalAccount)

	// HTMX Redirect to the subscribe request page.
	ctx.Response().Header().Set("HX-Redirect", forwardToURL)
	return ctx.String(http.StatusOK, "")
}
