package unsplash

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

func GetPhoto(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.unsplash.GetPhoto"

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Domain not found")
		}

		// Get the Giphy Provider and API Key
		connectionService := factory.Connection()
		unsplash := model.NewConnection()

		if err := connectionService.LoadByProvider(providers.ProviderTypeUnsplash, &unsplash); err != nil {
			return derp.Wrap(err, "handler.GetGiphyImages", "Giphy is not configured for this domain")
		}

		applicationName := unsplash.Data.GetString("applicationName")

		if applicationName == "" {
			return derp.NewNotFoundError(location, "Unsplash API ApplicationName cannot be empty", nil)
		}

		accessKey := unsplash.Data.GetString("accessKey")

		if accessKey == "" {
			return derp.NewNotFoundError(location, "Unsplash API AccessKey cannot be empty", nil)
		}

		photoID := ctx.Param("photo")

		if photoID == "" {
			return derp.NewBadRequestError(location, "Photo ID is required", nil)
		}

		asJSON := false

		if strings.HasSuffix(photoID, ".json") {
			photoID = strings.TrimSuffix(photoID, ".json")
			asJSON = true
		}

		photo, err := apiRequest(factory.HTTPCache(), accessKey, "/photos/"+photoID)

		if err != nil {
			return derp.Wrap(err, location, "Error getting photo from Unsplash API")
		}

		if asJSON {
			return ctx.JSONPretty(200, photo, "\t")
		}

		return displayPhoto(ctx, applicationName, photo)
	}
}

func GetCollectionRandom(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.unsplash.GetCollectionRandom"

	return func(ctx echo.Context) error {

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Domain not found")
		}

		// Get the Giphy Provider and API Key
		connectionService := factory.Connection()
		unsplash := model.NewConnection()

		if err := connectionService.LoadByProvider(providers.ProviderTypeUnsplash, &unsplash); err != nil {
			return derp.Wrap(err, "handler.GetGiphyImages", "Giphy is not configured for this domain")
		}

		applicationName := unsplash.Data.GetString("applicationName")

		if applicationName == "" {
			return derp.NewNotFoundError(location, "Unsplash API ApplicationName cannot be empty", nil)
		}

		accessKey := unsplash.Data.GetString("accessKey")

		if accessKey == "" {
			return derp.NewNotFoundError(location, "Unsplash API AccessKey cannot be empty", nil)
		}

		collectionID := ctx.Param("collection")

		if collectionID == "" {
			return derp.NewBadRequestError(location, "Photo ID is required", nil)
		}

		photo, err := apiRequest(factory.HTTPCache(), accessKey, "/photos/random?collections="+collectionID)

		if err != nil {
			return derp.Wrap(err, location, "Error getting photo from Unsplash API")
		}

		// If this iis a JSON request, then return nicely formatted JSON
		if asJSON := convert.Bool(ctx.QueryParam("json")); asJSON {
			return ctx.JSONPretty(200, photo, "\t")
		}

		// Otherwise, return the photo as HTML
		return displayPhoto(ctx, applicationName, photo)
	}
}

func apiRequest(httpCache *httpcache.HTTPCache, accessKey string, url string) (mapof.Any, error) {

	client := httpcache.NewHTTPClient(remote.DefaultClient(), httpCache)

	result := mapof.NewAny()
	txn := remote.Get("https://api.unsplash.com"+url).
		Accept("application/json").
		Header("Authorization", "Client-ID "+accessKey).
		Header("Accept-Version", "v1").
		Result(&result).
		UseClient(client)

	if err := txn.Send(); err != nil {
		return nil, err
	}

	spew.Dump("apiRequest.  This *SHOULD* have data in it now :(")
	httpCache.Debug()
	return result, nil
}

func displayPhoto(ctx echo.Context, applicationName string, photo mapof.Any) error {

	urls := photo.GetMap("urls")
	user := photo.GetMap("user")
	height := first.String(ctx.QueryParam("height"), "100%")
	width := first.String(ctx.QueryParam("width"), "100%")

	// UTM Trackers and Credits are required by Unsplash API
	tracker := "?utm_medium=referral&utm_source=" + url.QueryEscape(applicationName)
	credits := `Photo By <a href="https://unsplash.com/@` + user.GetString("username") + tracker + `" target="_blank">` +
		user.GetString("name") +
		`</a> on <a href="https://unsplash.com` + tracker + `" target="_blank">Unsplash</a>.&nbsp;`

	// Write the Unsplash HTML
	b := html.New()
	b.Picture().
		Style("height:"+height, "width:"+width, "object-fit:cover", "object-position:center center").
		EndBracket()

	b.Img(urls.GetString("full")).
		Attr("alt", photo.GetString("description")).
		Style("height:"+height, "width:"+width, "object-fit:cover", "object-position:center center").
		EndBracket()

	b.Close()
	b.Div().Class("text-gray text-xs").Style("text-align:right").InnerHTML(credits).Close()
	b.Close()

	return ctx.HTML(200, b.String())
}
