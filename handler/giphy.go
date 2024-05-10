package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

func GetGiphyWidget(serverFactory *server.Factory) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		sterankoContext := ctx.(*steranko.Context)

		// Verify authorization
		authorization := getAuthorization(sterankoContext)

		if !authorization.IsAuthenticated() {
			return derp.NewUnauthorizedError("handler.GetGiphyImages", "You must be logged in to use this feature")
		}

		// Get the factory for this domain
		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, "handler.GetGiphyImages", "Cannot load Domain")
		}

		// Get the Giphy Provider and API Key
		connectionService := factory.Connection()
		giphy := model.NewConnection()

		if err := connectionService.LoadByProvider(providers.ProviderTypeGiphy, &giphy); err != nil {
			return derp.Wrap(err, "handler.GetGiphyImages", "Giphy is not configured for this domain")
		}

		apiKey := giphy.Data.GetString(providers.Giphy_APIKey)

		b := html.New()

		b.Div().Style("position:absolute", "border:solid 1px black", "background-color:white", "max-height:150px", "overflow-y:scroll")
		b.Input("text", "").ID("giphySearch").Attr("placeholder", "Search Giphy").
			Script("on keyup log my value then log '" + apiKey + "'" +
				"set url to 'https://api.giphy.com/v1/gifs/search?api_key=" + apiKey + "&q=' & my value & '&limit=25&offset=0&rating=G&lang=en'" +
				"fetch https://api.giphy.com/v1/gifs/search ")
		b.CloseAll()

		return ctx.HTML(http.StatusOK, b.String())
	}
}
