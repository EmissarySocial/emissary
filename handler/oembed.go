package handler

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/oembed"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// oEmbedCacheAge is the number of seconds that consumers should cache an oEmbed document
const oEmbedCacheAge = 86400

// oEmbedThumbnailSize is the height and width (in pixels) of the thumbnail images we advertise
const oEmbedThumbnailSize = 300

// GetOEmbed will provide an OEmbed service to be used exclusively by websites on this domain.
//
// benpate/oembed owns the spec mechanics -- valid-by-construction documents, validation,
// and encoding -- while record resolution and access control stay here.  Request parsing
// also stays here: the library is a consumer library on that half, so its ParseRequest was
// cut deliberately and is not coming back.
//
// TODO: (oembed/TODO.md Phase 9.6) "maxwidth"/"maxheight" are still ignored; every document
// we serve is a "link" with a fixed 300px thumbnail.  This route is LIVE (bandwagon and
// qwertylicious templates advertise it via discovery links) and may not move without a
// compat alias.
func GetOEmbed(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetOEmbed"

	format := ctx.QueryParam("format") // nolint:scopeguard
	token := ctx.QueryParam("url")

	// RULE: The spec requires "501 Not Implemented" when a consumer demands a format we
	// cannot produce.  An empty value means "no preference" and is answered with JSON.
	// oembed.WriteResponse refuses the same set, so this guard is here to fail fast --
	// before spending a database lookup on a request we will never answer.
	switch format {

	case "", oembed.FormatJSON, oembed.FormatXML:
		// These are the formats we serve

	default:
		return derp.NotImplemented(location, "Unsupported oEmbed format", format)
	}

	// Parse the requested URL.  PrependProtocol lets a scheme-less value ("example.com/@user")
	// resolve too; without it, url.Parse reads the whole value as a path and finds neither
	// a hostname nor a record.
	parsedToken, err := url.Parse(uri.PrependProtocol(token))

	if err != nil {
		return derp.Wrap(err, location, "Invalid URL", token)
	}

	// RULE: oEmbed only describes records that live on THIS domain
	if notSameHostname(parsedToken.Host, factory.Hostname()) {
		return derp.NotFound(location, "URL does not match this domain", token)
	}

	// Load the oEmbed document for this URL
	result, err := getOEmbed_record(factory, session, parsedToken.Path)

	if err != nil {
		return derp.Wrap(err, location, "Loading OEmbed record", token)
	}

	// Serve the document in the requested format.  WriteResponse validates the document
	// before it writes anything, so we can never publish a spec-invalid document.
	if err := oembed.WriteResponse(ctx.Response(), result, format); err != nil {
		return derp.Wrap(err, location, "Writing oEmbed response", token)
	}

	// Embed, and let embed.
	return nil
}

// getOEmbed_record builds the oEmbed document that describes a single path on this domain
func getOEmbed_record(factory *service.Factory, session data.Session, path string) (oembed.Response, error) {

	token := oEmbedToken(path)

	// An empty path is this domain's home page
	if token == "" {
		return getOEmbed_Domain(factory), nil
	}

	// A leading "@" marks a User's profile
	if username, isUser := strings.CutPrefix(token, "@"); isUser {
		return getOEmbed_User(factory, session, username)
	}

	// Everything else names a Stream
	return getOEmbed_Stream(factory, session, token)
}

// getOEmbed_Domain builds the oEmbed document that describes this domain's home page
func getOEmbed_Domain(factory *service.Factory) oembed.Response {
	domain := factory.Domain().Get()
	return newOEmbedResponse(factory, domain.Label)
}

// getOEmbed_Stream builds the oEmbed document that describes a single Stream
func getOEmbed_Stream(factory *service.Factory, session data.Session, token string) (oembed.Response, error) {

	const location = "handler.getOEmbed_Stream"

	// Load the Stream
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByToken(session, token, &stream); err != nil {
		return oembed.Response{}, derp.Wrap(err, location, "Loading stream from database", token)
	}

	// RULE: Only expose oEmbed metadata for publicly-viewable Streams. oEmbed is
	// consumed anonymously by third-party sites, so a non-public Stream must not
	// leak its Label or thumbnail here. Return NotFound (not Forbidden) so we don't
	// confirm the token's existence.
	if !stream.DefaultAllowAnonymous() {
		return oembed.Response{}, derp.NotFound(location, "Stream not found", token)
	}

	// Describe the Stream
	result := newOEmbedResponse(factory, stream.Label)
	setOEmbedThumbnail(&result, factory.Hostname(), firstOf(stream.IconURL, stream.Data.GetString("bannerUrl")))

	return result, nil
}

// getOEmbed_User builds the oEmbed document that describes a single User's profile
func getOEmbed_User(factory *service.Factory, session data.Session, token string) (oembed.Response, error) {

	const location = "handler.getOEmbed_User"

	// Load the User
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByToken(session, token, &user); err != nil {
		return oembed.Response{}, derp.Wrap(err, location, "Loading user from database", token)
	}

	// RULE: Only expose oEmbed metadata for public User profiles. oEmbed is consumed
	// anonymously by third-party sites, so a non-public profile must not leak its
	// handle or avatar here. Return NotFound (not Forbidden) so we don't confirm the
	// token's existence.
	if !user.IsPublic {
		return oembed.Response{}, derp.NotFound(location, "User not found", token)
	}

	// Describe the User
	domain := factory.Domain().Get()
	result := newOEmbedResponse(factory, "@"+user.Username+"@"+domain.Hostname)
	setOEmbedThumbnail(&result, factory.Hostname(), user.ActivityPubIconURL())

	return result, nil
}

// newOEmbedResponse returns an oEmbed document carrying the values that every record on this domain shares
func newOEmbedResponse(factory *service.Factory, title string) oembed.Response {

	domain := factory.Domain().Get()

	// oembed.NewLink stamps the required version and type, so this document
	// cannot be spec-invalid by construction.
	result := oembed.NewLink(title)
	result.CacheAge = oEmbedCacheAge
	result.ProviderName = domain.Label
	result.ProviderURL = domain.Host()

	return result
}

// oEmbedToken reduces a URL path to the single token that identifies a record
func oEmbedToken(path string) string {

	// RULE: Only the FIRST path segment identifies a record.  This resolves every URL that
	// renders that record -- "/@user", "/@user/", "/@user/pub" -- to the same document,
	// instead of failing on a trailing slash or an action name.
	token, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")

	return token
}

// setOEmbedThumbnail adds the thumbnail fields to an oEmbed document, when an icon is available
func setOEmbedThumbnail(response *oembed.Response, hostname string, iconURL string) {

	// Records without an icon simply have no thumbnail
	if iconURL == "" {
		return
	}

	// RULE: The ".webp" extension and the height/width query are instructions to THIS
	// domain's mediaserver, so they are only added to URLs that this domain serves.
	// A remote icon (a federated Stream, or a hand-entered "bannerUrl") is published
	// as-is, because its own server cannot answer a resize request.
	if isLocalMediaURL(iconURL, hostname) {
		size := strconv.Itoa(oEmbedThumbnailSize)
		iconURL += ".webp?height=" + size + "&width=" + size
	}

	// SetThumbnail stamps url, width, and height together, so the spec's all-or-none
	// rule cannot be broken here.  A remote icon reports the size we asked for rather
	// than a size we measured.
	response.SetThumbnail(iconURL, oEmbedThumbnailSize, oEmbedThumbnailSize)
}

// isLocalMediaURL returns TRUE if a media URL can be resized by this domain's mediaserver
func isLocalMediaURL(value string, hostname string) bool {

	// A URL that already carries a query string cannot take a resize query, whatever its host
	if strings.Contains(value, "?") {
		return false
	}

	// A root-relative URL is always served by this domain
	if strings.HasPrefix(value, "/") {
		return true
	}

	// Otherwise, the URL must name this domain
	return !notSameHostname(value, hostname)
}

// notSameHostname returns TRUE if two values do NOT identify the same host on this server
func notSameHostname(value string, hostname string) bool {
	return normalizeHostname(value) != normalizeHostname(hostname)
}

// normalizeHostname reduces a URL (or a bare hostname) to a comparable hostname
func normalizeHostname(value string) string {

	// Mirrors server.factoryCore.normalizeHostname -- lower case, no port, no leading "www."
	// -- so that this endpoint accepts exactly the URLs that the request router accepts.
	// uri.Hostname already lower-cases the value and strips the protocol, path, and port.
	return strings.TrimPrefix(uri.Hostname(value), "www.")
}
