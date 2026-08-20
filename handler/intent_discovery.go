package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/convert"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// GetIntentInfo serves the discovery document that tells a remote site which Activity Intents this server supports
func GetIntentInfo(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetIntentInfo"

	// RULE: IntentType must not be empty
	if ctx.QueryParam("intent") == "" {
		return derp.BadRequest(location, "Intent must not be empty")
	}

	// Collect accountID
	accountID := ctx.QueryParam("account")

	if accountID == "" {
		return derp.BadRequest(location, "You must specify a Fediverse account")
	}

	// Look up the account via the ActivityService
	client := factory.ActivityStream().AppClient()
	actor, err := client.Load(accountID, sherlock.AsActor())

	if err != nil {
		return derp.Wrap(err, location, "Loading account from ActivityService")
	}

	// Return the account information to the client.
	ctx.Response().Header().Set("Hx-Push-Url", "false")

	// RULE: Every value below comes from a remote, attacker-controlled actor document (or the raw
	// accountID query param), and the client renders these fields by interpolating them into an HTML
	// string (insertAdjacentHTML) without escaping. So each value MUST be neutralized here, at the
	// source: SanitizeText strips all markup from the free-text display fields, and SafeURL rejects
	// non-http(s) schemes and percent-encodes any attribute-breakout characters in the URL fields.
	return ctx.JSON(http.StatusOK, mapof.Any{
		vocab.PropertyID:                uri.SafeURL(accountID),
		vocab.PropertyName:              convert.SanitizeText(actor.Name()),
		vocab.PropertyIcon:              uri.SafeURL(actor.Icon().Href()),
		vocab.PropertyURL:               uri.SafeURL(firstOf(actor.URL(), actor.ID())),
		vocab.PropertyPreferredUsername: convert.SanitizeText(ActorUsername(actor)),
	})
}
