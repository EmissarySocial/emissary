package unsplash

import (
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/httpcache"
	"github.com/benpate/color"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/html"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// GetPhoto returns a single photo from Unsplash. The user DOES NOT need to be authenticated.
func GetPhoto(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.unsplash.GetPhoto"

	// Get the Unsplash Provider and API Key
	connectionService := factory.Connection()
	unsplash := model.NewConnection()

	if err := connectionService.LoadByProvider(session, model.ConnectionProviderUnsplash, &unsplash); err != nil {
		return derp.Wrap(err, location, "Unsplash is not configured for this domain")
	}

	// Collect ApplicationName
	applicationName := unsplash.Data.GetString("applicationName")

	if applicationName == "" {
		return derp.NotFound(location, "Unsplash API ApplicationName cannot be empty", nil)
	}

	// Collect AccessKey
	accessKey := unsplash.Data.GetString("accessKey")

	if accessKey == "" {
		return derp.NotFound(location, "Unsplash API AccessKey cannot be empty", nil)
	}

	// Collect Photo ID
	photoID := ctx.Param("photo")

	if photoID == "" {
		return derp.BadRequest(location, "Photo ID is required", nil)
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
		return derp.Wrap(err, location, "Sending request to Unsplash API")
	}

	// If this is a JSON request, then return nicely formatted JSON
	if asJSON {
		return ctx.JSONPretty(200, photo, "\t")
	}

	// Otherwise, display the photo
	return displayPhoto(ctx, applicationName, photo)
}

// GetCollectionRandom returns a random photo from a collection on Unsplash. The user DOES NOT need to be authenticated.
func GetCollectionRandom(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.unsplash.GetCollectionRandom"

	// Get the Unsplash Provider and API Key
	connectionService := factory.Connection()
	unsplash := model.NewConnection()

	if err := connectionService.LoadByProvider(session, model.ConnectionProviderUnsplash, &unsplash); err != nil {
		return derp.Wrap(err, location, "Unsplash is not configured for this domain")
	}

	applicationName := unsplash.Data.GetString("applicationName")

	if applicationName == "" {
		return derp.NotFound(location, "Unsplash API ApplicationName cannot be empty", nil)
	}

	accessKey := unsplash.Data.GetString("accessKey")

	if accessKey == "" {
		return derp.NotFound(location, "Unsplash API AccessKey cannot be empty", nil)
	}

	collectionID := ctx.Param("collection")

	if collectionID == "" {
		return derp.BadRequest(location, "Photo ID is required", nil)
	}

	// Get the first 64 photos from the collection
	photos := make([]mapof.Any, 0, 64)

	txn := newTransaction(factory.HTTPCache(), accessKey).
		Get("https://api.unsplash.com/collections/" + collectionID + "/photos?per_page=64").
		Result(&photos)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Getting photo from Unsplash API")
	}

	if len(photos) == 0 {
		return derp.NotFound(location, "Collection is empty", collectionID)
	}

	// Select a random photo from the collection
	photo := photos[rand.Intn(len(photos))]

	// If this iis a JSON request, then return nicely formatted JSON
	if convert.Bool(ctx.QueryParam("json")) {
		return ctx.JSONPretty(200, photo, "\t")
	}

	// If this is a "forward" request, then redirect to the photo URL
	if convert.Bool(ctx.QueryParam("forward")) {
		target := photo.GetMap("urls").GetString("regular")

		// SECURITY: the redirect target comes from the Unsplash API response (external).
		// Legitimate values are always https image URLs on an unsplash.com host, so refuse
		// to forward anything else instead of acting as an open-redirect primitive.
		if !isUnsplashImageURL(target) {
			return derp.NotFound(location, "Unsplash API returned an invalid photo URL", target)
		}

		return ctx.Redirect(http.StatusSeeOther, target)
	}

	// Otherwise, return the photo as HTML
	return displayPhoto(ctx, applicationName, photo)
}

// isUnsplashImageURL reports whether target is a well-formed https URL on an unsplash.com
// host. It bounds the ?forward redirect to values the Unsplash API can legitimately return,
// rejecting empty, non-https, or off-site URLs so the endpoint cannot be used as an open redirect.
func isUnsplashImageURL(target string) bool {

	parsed, err := url.Parse(target)

	if err != nil {
		return false
	}

	if parsed.Scheme != "https" {
		return false
	}

	// Accept "unsplash.com" itself and any subdomain (e.g. "images.unsplash.com"),
	// but not look-alikes such as "unsplash.com.evil.example".
	host := parsed.Hostname()
	return host == "unsplash.com" || strings.HasSuffix(host, ".unsplash.com")
}

// newTransaction returns a cached, authenticated request to the Unsplash API
func newTransaction(cache *httpcache.HTTPCache, accessKey string) *remote.Transaction {

	return remote.New().
		With(options.WithRoundTripper(httpcache.NewHTTPMiddleware(cache))).
		Accept("application/json").
		Header("Authorization", "Client-ID "+accessKey).
		Header("Accept-Version", "v1")
}

// displayPhoto renders an Unsplash photo as HTML, including the attribution that the API requires
func displayPhoto(ctx echo.Context, applicationName string, photo mapof.Any) error {

	urls := photo.GetMap("urls")
	user := photo.GetMap("user")
	height := first.String(ctx.QueryParam("height"), "100%")
	width := first.String(ctx.QueryParam("width"), "100%")
	photoColor := photo.GetString("color")
	textColor := color.Parse(photoColor).Text().Hex()

	// UTM Trackers and Credits are required by Unsplash API
	tracker := "?utm_medium=referral&utm_source=" + url.QueryEscape(applicationName)

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

	// SECURITY: the photographer username/name come from the Unsplash API (external, untrusted).
	// Build the credit line with the html builder so the href is Attr-escaped and the display name is
	// InnerText-escaped, instead of hand-assembling an HTML string and emitting it via InnerHTML -- which
	// would let a crafted username/name inject markup that runs in the Emissary origin.
	b.Div().Class("pos-absolute-bottom-right padding-xs text-xs").Style("background-color:"+photoColor, "color:"+textColor).EndBracket()
	b.WriteString("Photo By ")
	b.A("https://unsplash.com/@"+user.GetString("username")+tracker).Attr("target", "_blank").Style("color:" + textColor).InnerText(user.GetString("name")).Close()
	b.WriteString(" on ")
	b.A("https://unsplash.com"+tracker).Attr("target", "_blank").Style("color:" + textColor).InnerText("Unsplash").Close()
	b.WriteString(".&nbsp;")
	b.Close()
	b.Close()

	return ctx.HTML(200, b.String())
}
