package handler

import (
	"net/http"

	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/html"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/render"
	"github.com/whisperverse/whisperverse/server"
)

func ListSubscriptions(fm *server.Factory) echo.HandlerFunc {

	const location = "handler.ListSubscriptions"

	return func(ctx echo.Context) error {

		// Try to load required services
		factory, err := fm.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Cannot find Domain")
		}

		subscriptionService := factory.Subscription()

		// Try to query the database
		it, err := subscriptionService.List(exp.All(), option.SortDesc("url"))

		if err != nil {
			return derp.Wrap(err, location, "Cannot retrieve subscriptions")
		}

		subscription := model.NewSubscription()

		// Build the HTML response
		b := html.New()

		b.Div()
		for it.Next(&subscription) {
			b.Div().InnerHTML(subscription.URL).Close()
			subscription = model.NewSubscription()
		}
		b.Close()

		// Write modal form to client
		result := render.WrapModalForm(ctx.Response(), "/subscriptions", b.String())
		return ctx.HTML(http.StatusOK, result)
	}
}

func GetSubscription(fm *server.Factory) echo.HandlerFunc {

	// const location = "handler.GetSubscription"

	return func(ctx echo.Context) error {

		return nil
	}
}

func PostSubscription(fm *server.Factory) echo.HandlerFunc {

	// const location = "handler.PostSubscription"

	return func(ctx echo.Context) error {
		return nil
	}
}

func DeleteSubscription(fm *server.Factory) echo.HandlerFunc {

	// const location = "handler.DeleteSubscription"

	return func(ctx echo.Context) error {
		return nil
	}
}
