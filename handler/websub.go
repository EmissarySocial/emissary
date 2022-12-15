package handler

import (
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	client "github.com/benpate/websub-client"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

func WebSubClient(serverFactory *server.Factory) echo.HandlerFunc {
	return func(ctx echo.Context) error {

		spew.Dump("A")
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handler.WebSubClient", "Error loading server factory")
		}

		spew.Dump("B")

		followingService := factory.Following()

		c := client.New(followingService.CallbackURL())

		c.AddHandler(func(event *client.SubscriptionDenied) {
			spew.Dump(event)
		})

		c.AddHandler(func(event *client.Publish) {
			spew.Dump(event)
		})

		c.ServeHTTP(ctx.Response(), ctx.Request())
		spew.Dump("D")

		return nil
	}
}
