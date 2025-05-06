package unsplash

import (
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/benpate/color"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
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
			return derp.Wrap(err, location, "Giphy is not configured for this domain")
		}

		applicationName := unsplash.Data.GetString("applicationName")

		if applicationName == "" {
			return derp.NotFoundError(location, "Unsplash API ApplicationName cannot be empty", nil)
		}

		accessKey := unsplash.Data.GetString("accessKey")

		if accessKey == "" {
			return derp.NotFoundError(location, "Unsplash API AccessKey cannot be empty", nil)
		}

		photoID := ctx.Param("photo")

		if photoID == "" {
			return derp.BadRequestError(location, "Photo ID is required", nil)
		}

		asJSON := false

		if strings.HasSuffix(photoID, ".json") {
			photoID = strings.TrimSuffix(photoID, ".json")
			asJSON = true
		}

		// Get the photo from the Unsplash API
		photo := mapof.NewAny()
		txn := newTransaction(factory.HTTPCache(), accessKey).
			Get("https://api.unsplash.com/photos/" + photoID).
			Result(&photo)

		if err := txn.Send(); err != nil {
			return derp.Wrap(err, location, "Error sending request to Unsplash API")
		}

		// If this is a JSON request, then return nicely formatted JSON
		if asJSON {
			return ctx.JSONPretty(200, photo, "\t")
		}

		// Otherwise, display the photo
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
			return derp.Wrap(err, "handler.unsplash.GetCollectionRandom", "Giphy is not configured for this domain")
		}

		applicationName := unsplash.Data.GetString("applicationName")

		if applicationName == "" {
			return derp.NotFoundError(location, "Unsplash API ApplicationName cannot be empty", nil)
		}

		accessKey := unsplash.Data.GetString("accessKey")

		if accessKey == "" {
			return derp.NotFoundError(location, "Unsplash API AccessKey cannot be empty", nil)
		}

		collectionID := ctx.Param("collection")

		if collectionID == "" {
			return derp.BadRequestError(location, "Photo ID is required", nil)
		}

		// Get the first 64 photos from the collection
		photos := make([]mapof.Any, 0, 64)

		txn := newTransaction(factory.HTTPCache(), accessKey).
			Get("https://api.unsplash.com/collections/" + collectionID + "/photos?per_page=64").
			Result(&photos)

		if err := txn.Send(); err != nil {
			return derp.Wrap(err, location, "Error getting photo from Unsplash API")
		}

		if len(photos) == 0 {
			return derp.NotFoundError(location, "Collection is empty", collectionID)
		}

		// Select a random photo from the collection
		photo := photos[rand.Intn(len(photos))]

		// If this iis a JSON request, then return nicely formatted JSON
		if convert.Bool(ctx.QueryParam("json")) {
			return ctx.JSONPretty(200, photo, "\t")
		}

		// If this is a "forward" request, then redirect to the photo URL
		if convert.Bool(ctx.QueryParam("forward")) {
			url := photo.GetMap("urls").GetString("regular")
			return ctx.Redirect(http.StatusSeeOther, url)
		}

		// Otherwise, return the photo as HTML
		return displayPhoto(ctx, applicationName, photo)
	}
}

func newTransaction(cache *httpcache.HTTPCache, accessKey string) *remote.Transaction {

	return remote.New().
		Client(httpcache.NewHTTPClient(cache)).
		Accept("application/json").
		Header("Authorization", "Client-ID "+accessKey).
		Header("Accept-Version", "v1")
}

func displayPhoto(ctx echo.Context, applicationName string, photo mapof.Any) error {

	urls := photo.GetMap("urls")
	user := photo.GetMap("user")
	height := first.String(ctx.QueryParam("height"), "100%")
	width := first.String(ctx.QueryParam("width"), "100%")
	photoColor := photo.GetString("color")
	textColor := color.Parse(photoColor).Text().Hex()

	// UTM Trackers and Credits are required by Unsplash API
	tracker := "?utm_medium=referral&utm_source=" + url.QueryEscape(applicationName)
	credits := `Photo By <a href="https://unsplash.com/@` + user.GetString("username") + tracker + `" target="_blank" style="color:` + textColor + `">` +
		user.GetString("name") +
		`</a> on <a href="https://unsplash.com` + tracker + `" target="_blank" style="color:` + textColor + `">Unsplash</a>.&nbsp;`

	// Write the Unsplash HTML
	b := html.New()
	b.Picture().
		Style("height:"+height, "width:"+width, "object-fit:cover", "object-position:center center").
		EndBracket()

	b.Source().SrcSet(urls.GetString("regular")).Media("(max-width:1080)").Close()
	b.Source().SrcSet(urls.GetString("small")).Media("(max-width:400px)").Close()

	b.Img(urls.GetString("regular")).
		Attr("alt", photo.GetString("alt_description")).
		Style("height:"+height, "width:"+width, "object-fit:cover", "object-position:center center").
		EndBracket()

	b.Close()
	b.Div().Class("pos-absolute-bottom-right padding-xs text-xs").Style("background-color:"+photoColor, "color:"+textColor).InnerHTML(credits).Close()
	b.Close()

	return ctx.HTML(200, b.String())
}
