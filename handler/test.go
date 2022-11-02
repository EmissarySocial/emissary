package handler

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/maps"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

func TestTwitter(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.Test"

	return func(ctx echo.Context) error {

		// Try to get domain Factory for this hostname
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error getting factory")
		}

		// Try to locate the existing Domain
		domainService := factory.Domain()
		domain := model.NewDomain()

		if err := domainService.Load(&domain); err != nil {
			return derp.Wrap(err, location, "Error loading domain")
		}

		client, _ := domain.Clients.Get(model.ProviderTwitter)

		if !client.Active {
			return derp.NewInternalError(location, "Twitter client is not active")
		}

		success := maps.New()
		failure := maps.New()

		txn := remote.Get("https://api.twitter.com/2/users/me").
			Header("Authorization", client.Token.TokenType+" "+client.Token.AccessToken).
			Response(&success, &failure)

		if err := txn.Send(); err != nil {
			return derp.NewInternalError(location, "Error sending request", err)
		}

		spew.Dump(success, failure)

		spew.Dump("here")
		return nil
	}
}
