package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/sherlock"
	"github.com/labstack/echo/v4"
)

func GetIntentInfo(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetIntentInfo"

	return func(ctx echo.Context) error {

		// Collect intentType
		intentType := ctx.QueryParam("intent")

		if intentType == "" {
			return derp.NewBadRequestError(location, "You must specify an intent")
		}

		// Collect accountID
		accountID := ctx.QueryParam("account")

		if accountID == "" {
			return derp.NewBadRequestError(location, "You must specify a Fediverse account")
		}

		// Get the domain factory
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error getting server factory by context")
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
			vocab.PropertyID:   actor.ID(),
			vocab.PropertyName: actor.Name(),
			vocab.PropertyIcon: actor.Icon().Href(),
			vocab.PropertyURL:  actor.URL(),
		})
	}
}
