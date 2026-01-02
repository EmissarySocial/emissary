package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/steranko"
)

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
		return derp.Wrap(err, location, "Unable to load account from ActivityService")
	}

	// Return the account information to the client
	ctx.Response().Header().Set("Hx-Push-Url", "false")

	return ctx.JSON(http.StatusOK, mapof.Any{
		vocab.PropertyID:                accountID,
		vocab.PropertyName:              actor.Name(),
		vocab.PropertyIcon:              actor.Icon().Href(),
		vocab.PropertyURL:               firstOf(actor.URL(), actor.ID()),
		vocab.PropertyPreferredUsername: ActorUsername(actor),
	})
}
