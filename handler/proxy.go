package handler

import (
	"net"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Proxy URL
 *
 * Signed-in Users may load any document, signed with their own key.
 *
 * Anonymous visitors may load actors only, signed by the application actor. CamperJS shows a
 * visitor's name and icon from their actor, and the visitor's account lives on another server,
 * so they cannot sign in here. Their browser cannot read the actor either: it holds no key to
 * sign with, Mastodon refuses unsigned actor fetches (mastodon.social today, and every Mastodon
 * server by default from 5.0), and some servers, such as pixelfed.social, send no CORS header.
 * Every anonymous failure answers 403, so that Digital Dome counts it against the caller.
 ******************************************/

// proxyUserAgentLimit caps the visitor's browser description forwarded to the remote server
const proxyUserAgentLimit = 512

// PostProxyURL is the handler for the "Proxy URL" endpoint, which loads a remote ActivityStreams
// document for the caller. Anonymous callers may load actors only.
// https://www.w3.org/TR/activitypub/#proxyUrl
func PostProxyURL(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.PostProxyURL"

	// Collect the URL to load. Each branch checks the error, because each answers it differently
	remoteURL, err := bindProxyURL(ctx, factory)

	// Signed-in Users load any document, signed with their own key
	if authorization := getAuthorization(ctx); authorization.IsAuthenticated() {

		if err != nil {
			return derp.Wrap(err, location, "Reading the requested URL")
		}

		return proxyForUser(ctx, factory, session, authorization.UserID, remoteURL)
	}

	// RULE: A visitor's bad URL is Forbidden too, so that Digital Dome counts it against the caller
	if err != nil {
		return derp.Wrap(err, location, "Reading the requested URL", derp.WithForbidden())
	}

	// Visitors whose accounts live on other servers may load actors only, because CamperJS needs
	// their actors and a browser cannot read them: it cannot sign the fetch Mastodon requires
	return proxyForVisitor(ctx, factory, remoteURL)
}

// bindProxyURL reads the requested URL from the request, refusing an empty, invalid, or local one
func bindProxyURL(ctx *steranko.Context, factory *service.Factory) (string, error) {

	const location = "handler.bindProxyURL"

	// Define the transaction we expect to receive.
	transaction := struct {
		URL string `form:"id" json:"id"`
	}{}

	// Bind the form data to the transaction object.
	if err := ctx.Bind(&transaction); err != nil {
		return "", derp.Wrap(err, location, "Binding form data")
	}

	// RULE: Don't allow empty IDs
	if transaction.URL == "" {
		return "", derp.Validation("Parameter 'id' is required")
	}

	// RULE: Remote value MUST be a valid URL
	if uri.NotValidURL(transaction.URL) {
		return "", derp.Validation("Parameter 'id' must be a valid URL", transaction.URL)
	}

	// Fast, friendly reject for obviously-local URLs. The authoritative SSRF
	// guard is the resolved-IP check in remote's transport.
	if uri.IsLocalURL(transaction.URL) {
		if uri.NotLocalHostname(factory.Hostname()) {
			return "", derp.Validation("Parameter 'id' must not be a local address", transaction.URL)
		}
	}

	return transaction.URL, nil
}

// proxyForUser loads any document for a signed-in User, signed with the User's key, and refreshes the cache
func proxyForUser(ctx *steranko.Context, factory *service.Factory, session data.Session, userID primitive.ObjectID, remoteURL string) error {

	const location = "handler.proxyForUser"

	// Load the signed-in User
	user := model.NewUser()

	if err := factory.User().LoadByID(session, userID, &user); err != nil {
		return derp.Wrap(err, location, "Loading User", derp.WithUnauthorized())
	}

	// RULE: A User who has moved to another server no longer signs requests from this one
	if user.MovedTo != "" {
		return derp.Unauthorized(location, "This User has moved to another server")
	}

	// Load the document from the URL (DON'T load from cache, but DO refresh the cache)
	client := factory.ActivityStream().UserClient(user.UserID)
	result, err := client.Load(remoteURL, ascache.WithWriteOnly())

	if err != nil {
		return derp.Wrap(err, location, "Loading URL", remoteURL)
	}

	// Success
	return ctx.JSON(http.StatusOK, result.Value())
}

// proxyForVisitor loads an actor for an anonymous caller, signed by the application actor, telling the
// remote server which address and browser asked for it, and answers Forbidden to anything else
func proxyForVisitor(ctx *steranko.Context, factory *service.Factory, remoteURL string) error {

	const location = "handler.proxyForVisitor"

	// Every failure here is Forbidden, whatever its cause, because Digital Dome counts only 403s
	// against a caller. WithForbidden replaces the cause's code; the cause stays for the error log.

	// Load the document, which may come from the cache
	clientIP := factory.ClientIP(ctx.Request())
	client := factory.ActivityStream().AppClient()
	result, err := client.Load(remoteURL, proxyVisitorHeaders(clientIP, ctx.Request().UserAgent()))

	if err != nil {
		return derp.Wrap(err, location, "Loading URL", remoteURL, derp.WithForbidden())
	}

	// RULE: Anonymous callers may load actors only. CamperJS needs nothing else, and anything wider
	// would let strangers fetch any document under this server's signature
	if result.NotActor() {
		return derp.Forbidden(location, "Document is not an actor", remoteURL)
	}

	// Success
	return ctx.JSON(http.StatusOK, result.Value())
}

// proxyVisitorHeaders returns a remote.Option that tells the remote server the anonymous visitor's
// address and browser. Emissary's own User-Agent stays, because it names the signing actor.
func proxyVisitorHeaders(clientIP string, userAgent string) remote.Option {

	// RULE: Forward only a real address, so the header never carries arbitrary text
	address := net.ParseIP(clientIP)

	// Truncate the browser description on a rune boundary
	userAgent = strings.ToValidUTF8(userAgent, "")
	if len(userAgent) > proxyUserAgentLimit {
		userAgent = strings.ToValidUTF8(userAgent[:proxyUserAgentLimit], "")
	}

	return remote.Option{
		ModifyRequest: func(_ *remote.Transaction, request *http.Request) *http.Response {

			// Name the visitor's address (RFC 7239), quoting it and bracketing IPv6
			if address != nil {
				if address.To4() != nil {
					request.Header.Set("Forwarded", `for="`+address.String()+`"`)
				} else {
					request.Header.Set("Forwarded", `for="[`+address.String()+`]"`)
				}
				request.Header.Set("X-Forwarded-For", address.String())
			}

			// Name the visitor's browser, which no standard header carries
			if userAgent != "" {
				request.Header.Set("X-Forwarded-User-Agent", userAgent)
			}

			// Send the request as normal
			return nil
		},
	}
}
