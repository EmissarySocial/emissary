package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/benpate/steranko"
)

func GetIntentInfo(ctx *steranko.Context, factory *domain.Factory, session data.Session) error {

	const location = "handler.GetIntentInfo"

	// Collect intentType
	intentType := ctx.QueryParam("intent")

	if intentType == "" {
		return derp.BadRequestError(location, "You must specify an intent")
	}

	// Collect accountID
	accountID := ctx.QueryParam("account")

	if accountID == "" {
		return derp.BadRequestError(location, "You must specify a Fediverse account")
	}

	// Look up the account via the ActivityService
	activityService := factory.ActivityStream()
	actor, err := activityService.Load(accountID, sherlock.AsActor())

	if err != nil {
		return derp.Wrap(err, location, "Error loading account from ActivityService")
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
