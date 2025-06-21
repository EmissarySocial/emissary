package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetOutboundIntent translates an intent+account into a URL where we forward the
// user to complete the intent on their home server.
func GetOutboundIntent(ctx *steranko.Context, factory *domain.Factory) error {

	// Load the template for this account/intent
	camper := factory.Camper()
	intent := ctx.Param("intent")
	account := ctx.QueryParam("account")
	template := camper.GetTemplate(intent, account)

	if template == "" {
		return outboundIntentError(ctx, intent)
	}

	// Append success/cancel handlers to the URL
	data := ctx.QueryParams()
	data.Set("on-success", "(close)")
	data.Set("on-cancel", "(close)")

	// Populate the template with data from the stream
	nextURL := camper.PopulateTemplate(template, data)

	// Forward the user to the correct URL on their home server
	return ctx.Redirect(http.StatusTemporaryRedirect, nextURL)
}

func outboundIntentError(ctx echo.Context, intent string) error {

	b := html.New()

	b.Div()
	b.H1().InnerText("Sorry, this can't be completed")
	b.H2().InnerText("Your home server doesn't support the '" + intent + "' action.").Close()
	b.Div()
	b.Button().Attr("onclick", "window.close()").InnerText("Close Window")
	b.CloseAll()

	return ctx.HTML(http.StatusOK, b.String())
}
