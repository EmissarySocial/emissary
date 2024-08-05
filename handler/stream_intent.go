package handler

import (
	"net/http"
	"net/url"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/labstack/echo/v4"
)

func GetStreamIntent(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetStreamIntent"

	return func(ctx echo.Context) error {

		// Look up the domain factory
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Load the template for this account/intent
		camper := factory.Camper()
		intent := ctx.Param("intent")
		account := ctx.QueryParam("account")
		template := camper.GetTemplate(intent, account)

		if template == "" {
			return getStreamIntentError(ctx, account, intent)
		}

		// Try to load the stream from the database
		streamService := factory.Stream()
		stream := model.NewStream()
		streamToken := ctx.Param("stream")

		if err := streamService.LoadByToken(streamToken, &stream); err != nil {
			return derp.Wrap(err, location, "Error loading stream")
		}

		// Populate the template with data from the stream
		nextURL := camper.PopulateTemplate(template, ctx.Request().URL.Query())

		closeURL := url.QueryEscape(factory.Hostname() + "/.close-window")
		nextURL += "&on-success=" + closeURL + "&on-cancel=" + closeURL

		// Forward the user to the correct URL on their home server
		return ctx.Redirect(http.StatusTemporaryRedirect, nextURL)
	}
}

func getStreamIntentError(ctx echo.Context, account string, intent string) error {

	b := html.New()

	b.Div()
	b.H1().InnerText("Sorry, this can't be completed")
	b.H2().InnerText("Your home server doesn't support the '" + intent + "' action.").Close()
	b.Div()
	b.Button().Attr("onclick", "window.close()").InnerText("Close Window")
	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}

func GetCloseWindow(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {
		return ctx.HTML(http.StatusOK, `<html><head><script>window.close();</script></head></html>`)
	}
}
