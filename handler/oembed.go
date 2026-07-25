package handler

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/benpate/uri"
)

// oEmbedVersion is the version of the oEmbed specification that these documents conform to
const oEmbedVersion = "1.0"

// oEmbedTypeLink is the oEmbed document type used for records that have no embeddable player
const oEmbedTypeLink = "link"

// oEmbedCacheAge is the number of seconds that consumers should cache an oEmbed document
const oEmbedCacheAge = 86400

// oEmbedThumbnailSize is the height and width (in pixels) of the thumbnail images we advertise
const oEmbedThumbnailSize = 300

// oEmbedResponse is the response document defined by the oEmbed specification (https://oembed.com)
type oEmbedResponse struct {

	// This MUST remain a struct, not a map.  encoding/xml cannot marshal a map at all, so
	// a map-based document makes every "format=xml" request fail before it writes a byte.

	XMLName         xml.Name `json:"-"                          xml:"oembed"`
	Version         string   `json:"version"                    xml:"version"`
	Type            string   `json:"type"                       xml:"type"`
	Title           string   `json:"title,omitempty"            xml:"title,omitempty"`
	CacheAge        int      `json:"cache_age"                  xml:"cache_age"`
	ProviderName    string   `json:"provider_name"              xml:"provider_name"`
	ProviderURL     string   `json:"provider_url"               xml:"provider_url"`
	ThumbnailURL    string   `json:"thumbnail_url,omitempty"    xml:"thumbnail_url,omitempty"`
	ThumbnailHeight int      `json:"thumbnail_height,omitempty" xml:"thumbnail_height,omitempty"`
	ThumbnailWidth  int      `json:"thumbnail_width,omitempty"  xml:"thumbnail_width,omitempty"`
}

// GetOEmbed will provide an OEmbed service to be used exclusively by websites on this domain.
func GetOEmbed(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetOEmbed"

	format := ctx.QueryParam("format") // nolint:scopeguard
	token := ctx.QueryParam("url")

	// RULE: The spec requires "501 Not Implemented" when a consumer demands a format we
	// cannot produce.  An empty value means "no preference" and is answered with JSON.
	switch format {

	case "", "json", "xml":
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

	// Serve the document in the requested format
	if format == "xml" {
		return ctx.XML(http.StatusOK, result)
	}

	return ctx.JSON(http.StatusOK, result)
}

// getOEmbed_record builds the oEmbed document that describes a single path on this domain
func getOEmbed_record(factory *service.Factory, session data.Session, path string) (oEmbedResponse, error) {

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
func getOEmbed_Domain(factory *service.Factory) oEmbedResponse {
	domain := factory.Domain().Get()
	return newOEmbedResponse(factory, domain.Label)
}

// getOEmbed_Stream builds the oEmbed document that describes a single Stream
func getOEmbed_Stream(factory *service.Factory, session data.Session, token string) (oEmbedResponse, error) {

	const location = "handler.getOEmbed_Stream"

	// Load the Stream
	streamService := factory.Stream()
	stream := model.NewStream()

	if err := streamService.LoadByToken(session, token, &stream); err != nil {
		return oEmbedResponse{}, derp.Wrap(err, location, "Loading stream from database", token)
	}

	// RULE: Only expose oEmbed metadata for publicly-viewable Streams. oEmbed is
	// consumed anonymously by third-party sites, so a non-public Stream must not
	// leak its Label or thumbnail here. Return NotFound (not Forbidden) so we don't
	// confirm the token's existence.
	if !stream.DefaultAllowAnonymous() {
		return oEmbedResponse{}, derp.NotFound(location, "Stream not found", token)
	}

	// Describe the Stream
	result := newOEmbedResponse(factory, stream.Label)
	result.setThumbnail(factory.Hostname(), firstOf(stream.IconURL, stream.Data.GetString("bannerUrl")))

	return result, nil
}

// getOEmbed_User builds the oEmbed document that describes a single User's profile
func getOEmbed_User(factory *service.Factory, session data.Session, token string) (oEmbedResponse, error) {

	const location = "handler.getOEmbed_User"

	// Load the User
	userService := factory.User()
	user := model.NewUser()

	if err := userService.LoadByToken(session, token, &user); err != nil {
		return oEmbedResponse{}, derp.Wrap(err, location, "Loading user from database", token)
	}

	// RULE: Only expose oEmbed metadata for public User profiles. oEmbed is consumed
	// anonymously by third-party sites, so a non-public profile must not leak its
	// handle or avatar here. Return NotFound (not Forbidden) so we don't confirm the
	// token's existence.
	if !user.IsPublic {
		return oEmbedResponse{}, derp.NotFound(location, "User not found", token)
	}

	// Describe the User
	domain := factory.Domain().Get()
	result := newOEmbedResponse(factory, "@"+user.Username+"@"+domain.Hostname)
	result.setThumbnail(factory.Hostname(), user.ActivityPubIconURL())

	return result, nil
}

// newOEmbedResponse returns an oEmbed document carrying the values that every record on this domain shares
func newOEmbedResponse(factory *service.Factory, title string) oEmbedResponse {

	domain := factory.Domain().Get()

	return oEmbedResponse{
		Version:      oEmbedVersion,
		Type:         oEmbedTypeLink,
		Title:        title,
		CacheAge:     oEmbedCacheAge,
		ProviderName: domain.Label,
		ProviderURL:  domain.Host(),
	}
}

// oEmbedToken reduces a URL path to the single token that identifies a record
func oEmbedToken(path string) string {

	// RULE: Only the FIRST path segment identifies a record.  This resolves every URL that
	// renders that record -- "/@user", "/@user/", "/@user/pub" -- to the same document,
	// instead of failing on a trailing slash or an action name.
	token, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")

	return token
}

// setThumbnail adds the thumbnail fields to an oEmbed document, when an icon is available
func (response *oEmbedResponse) setThumbnail(hostname string, iconURL string) {

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

	// The spec requires dimensions alongside every thumbnail_url, so a remote icon
	// reports the size we asked for rather than a size we measured.
	response.ThumbnailURL = iconURL
	response.ThumbnailHeight = oEmbedThumbnailSize
	response.ThumbnailWidth = oEmbedThumbnailSize
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
