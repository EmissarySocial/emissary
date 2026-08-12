package service

import (
	"crypto"
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/hannibal/outbox"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Locator is used to find objects based on their URL or WebFinger token
type Locator struct {
	domainService       *Domain
	searchDomainService *SearchDomain
	searchQueryService  *SearchQuery
	streamService       *Stream
	userService         *User
	host                string
}

// NewLocator returns a fully initialized Locator service
func NewLocator() Locator {
	return Locator{}
}

// Refresh updates any stateful data that is cached inside this service.
func (service *Locator) Refresh(factory *Factory) {

	service.domainService = factory.Domain()
	service.searchDomainService = factory.SearchDomain()
	service.streamService = factory.Stream()
	service.searchQueryService = factory.SearchQuery()
	service.userService = factory.User()
	service.host = factory.Host()
}

// GetWebFingerResult returns a digit.Resource object based on the provided resource string.
func (service *Locator) GetWebFingerResult(session data.Session, resource string) (digit.Resource, error) {

	const location = "service.Locator.GetWebFingerResult"

	objectType, token := locateObjectFromURL(service.host, resource) // nolint:scopeguard

	switch objectType {

	case model.ActorTypeApplication:
		return service.domainService.WebFinger(), nil

	case model.ActorTypeSearchDomain:
		return service.searchDomainService.WebFinger(), nil

	case model.ActorTypeSearchQuery:
		return service.searchQueryService.WebFinger(session, token)

	case model.ActorTypeStream:
		return service.streamService.WebFinger(session, token)

	case model.ActorTypeUser:
		return service.userService.WebFinger(session, token)

	}

	// RULE: An unrecognized object type means the resource named another host, or named nothing we
	// publish. Both are 404s (RFC 7033 Section 4.5): the request itself is well formed, this server
	// is simply not authoritative for what it asked about.
	return digit.Resource{}, derp.NotFound(location, "Unknown WebFinger resource", resource)
}

// GetObjectFromURL parses a URL and verifies the existence of the referenced object.
func (service *Locator) GetObjectFromURL(session data.Session, value string) (string, primitive.ObjectID, error) {

	const location = "service.Locator.GetObjectFromURL"

	objectType, token := locateObjectFromURL(service.host, value)

	// Verify database records
	switch objectType {

	case model.ActorTypeApplication:
		return "", primitive.NilObjectID, derp.BadRequest(location, "Invalid Object Type", objectType)

	case model.ActorTypeSearchDomain:
		return "", primitive.NilObjectID, derp.BadRequest(location, "Invalid Object Type", objectType)

	case model.ActorTypeSearchQuery:
		return "", primitive.NilObjectID, derp.BadRequest(location, "Invalid Object Type", objectType)

	case model.ActorTypeStream:

		stream := model.NewStream()

		if err := service.streamService.LoadByToken(session, token, &stream); err != nil {
			return "", primitive.NilObjectID, derp.Wrap(err, location, "Loading stream", token)
		}

		return model.ActorTypeStream, stream.StreamID, nil

	case model.ActorTypeUser:

		user := model.NewUser()

		if err := service.userService.LoadByToken(session, token, &user); err != nil {
			return "", primitive.NilObjectID, derp.Wrap(err, location, "Loading user", token)
		}

		return model.ActorTypeUser, user.UserID, nil
	}

	// Fall through is failure.  Feel bad.
	return "", primitive.NilObjectID, derp.BadRequest(location, "Invalid Object Type", objectType)
}

// GetActor returns the outbox Actor for the named actor type and ID
func (service *Locator) GetActor(session data.Session, actorType string, actorID string) (outbox.Actor, error) {

	const location = "service.Locator.GetActor"

	switch actorType {

	case model.ActorTypeApplication:
		return service.domainService.ActivityPubActor(session)

	case model.ActorTypeSearchDomain:
		return service.searchDomainService.ActivityPubActor(session)

	case model.ActorTypeSearchQuery:
		if searchQueryID, err := primitive.ObjectIDFromHex(actorID); err == nil {
			return service.searchQueryService.ActivityPubActor(session, searchQueryID)
		}

	case model.ActorTypeStream:
		if streamID, err := primitive.ObjectIDFromHex(actorID); err == nil {
			return service.streamService.ActivityPubActor(session, streamID)
		}

	case model.ActorTypeUser:
		if userID, err := primitive.ObjectIDFromHex(actorID); err == nil {
			return service.userService.ActivityPubActor(session, userID)
		}

	default:
		return outbox.Actor{}, derp.BadRequest(location, "Invalid Actor Type", actorType)
	}

	return outbox.Actor{}, derp.BadRequest(location, "ActorID must be a valid ObjectID", actorType)
}

// PublicKeyID returns the keyId that the provided LOCAL actor advertises in its actor document,
// and which it must therefore use when signing. Every branch delegates to the same accessor that
// builds the actor document, so the signed and published identifiers cannot drift apart.
//
// This is deliberately separate from the private-key lookup in GetPrivateKey: several actors SHARE
// one private key, but no two actors may share a keyId. Receivers that bind the HTTP Signature to
// the Activity's actor -- hannibal (validator.HTTPSig.Validate) and Mastodon -- reject a keyId
// belonging to a different actor, even when the key material itself is valid.
//
// It reads no database, so the routing is unit-testable on its own.
func (service *Locator) PublicKeyID(actorType string, actorID primitive.ObjectID) (string, error) {

	const location = "service.locator.PublicKeyID"

	switch actorType {

	case model.ActorTypeApplication:
		return service.domainService.PublicKeyID(), nil

	case model.ActorTypeSearchDomain:
		return service.searchDomainService.PublicKeyID(), nil

	case model.ActorTypeSearchQuery:

		// A SearchQuery keyId is derived from its ID, so a missing ID would mint a keyId for an
		// actor that does not exist ("@search_000000000000000000000000#main-key"). Fail loudly
		// instead -- unlike the two Domain-level actors above, the ID is not optional here.
		if actorID.IsZero() {
			return "", derp.BadRequest(location, "SearchQuery actorID cannot be empty", actorType)
		}

		return service.searchQueryService.PublicKeyID(actorID), nil

	case model.ActorTypeStream:
		return service.streamService.PublicKeyID(actorID), nil

	case model.ActorTypeUser:
		return service.userService.PublicKeyID(actorID), nil
	}

	return "", derp.BadRequest(location, "Invalid Actor Type", actorType)
}

// GetPrivateKey returns the key ID and private key that the named Actor signs with
func (service *Locator) GetPrivateKey(session data.Session, actorType string, actorID primitive.ObjectID) (publicKeyID string, privateKey crypto.PrivateKey, err error) {

	const location = "service.locator.GetPrivateKey"

	// The advertised keyId is this actor's own, whatever key material it signs with.
	publicKeyID, err = service.PublicKeyID(actorType, actorID)

	if err != nil {
		return "", nil, derp.Wrap(err, location, "Locating public key ID", "actorType", actorType)
	}

	switch actorType {

	// RULE: These three actors SHARE the Domain private key. Sharing key MATERIAL across actors is
	// fine -- each one publishes that same public key in its own document. Sharing a keyId is not,
	// which is why the identifier above is resolved per-actor.
	case model.ActorTypeApplication,
		model.ActorTypeSearchDomain,
		model.ActorTypeSearchQuery:

		privateKey, err := service.domainService.PrivateKey(session)
		return publicKeyID, privateKey, err

	case model.ActorTypeStream:
		privateKey, err := service.streamService.PrivateKey(session, actorID)
		return publicKeyID, privateKey, err

	case model.ActorTypeUser:
		privateKey, err := service.userService.PrivateKey(session, actorID)
		return publicKeyID, privateKey, err
	}

	return "", nil, derp.BadRequest(location, "Invalid Actor Type", actorType)
}

// locateObjectFromURL parses a URL or WebFinger account, determines what type of object it is, and
// extracts the objectID.  It requires the current host (protocol + domain) to match. The returned
// object type can be one of: (Stream, User, SearchQuery, SearchDomain, or Application).  If the
// object is not found -- including when the value names a DIFFERENT host -- then both the type and
// token will be empty strings.
func locateObjectFromURL(host string, value string) (string, string) {

	hostname := uri.Hostname(host)

	// It's all good, bro. We're gonna deviate from the spec,
	// and just NOT CARE if you include `acct:` or not.
	value = strings.TrimPrefix(value, "acct:")

	// Values that carry a protocol are URLs ("https://example.com/@username"); everything else is a
	// WebFinger account ("@username@example.com") or a naked username ("username").
	if strings.Contains(value, uri.ProtocolSuffix) {
		return locateObjectFromPath(hostname, value)
	}

	return locateObjectFromAccount(hostname, value)
}

// locateObjectFromPath identifies the object named by a URL-type value, for example
// "https://example.com/@username".
func locateObjectFromPath(hostname string, value string) (string, string) {

	// RULE: A URL must name THIS host. Compare the bare hostname -- uri.Hostname folds case and
	// strips the port, per RFC 3986 -- because a domain reached on a non-standard port still serves
	// the same objects.
	if uri.Hostname(value) != hostname {
		return "", ""
	}

	// Parsing (rather than trimming the host prefix) discards the query string and fragment, and
	// tolerates a protocol that does not match the one this server advertises.
	parsedURL, err := url.Parse(value)

	if err != nil {
		return "", ""
	}

	path := strings.TrimPrefix(parsedURL.Path, "/")

	// Special case for "Application" account
	if path == "" {
		return model.ActorTypeApplication, ""
	}

	// Keep only the first path segment; any trailing route data is discarded
	// (e.g. "token/route" and "token/" both resolve on "token").
	path, _, _ = strings.Cut(path, "/")

	// Special case for "Application" account
	if path == "@application" {
		return model.ActorTypeApplication, ""
	}

	// Identify Global Search actor
	if path == "@search" {
		return model.ActorTypeSearchDomain, ""
	}

	// Identify SearchQuery URLs
	if searchID, found := strings.CutPrefix(path, "@search_"); found {
		return model.ActorTypeSearchQuery, searchID
	}

	// Identify User URLs
	if userID, found := strings.CutPrefix(path, "@"); found {
		return model.ActorTypeUser, userID
	}

	// Trim off any trailing path data
	return model.ActorTypeStream, path
}

// locateObjectFromAccount identifies the object named by an account-type value, for example
// "@username@example.com", "username@example.com", or a naked "username".
func locateObjectFromAccount(hostname string, value string) (string, string) {

	// Remove the leading "@" (if present) so that the only "@" that can remain is the one that
	// separates the username from its host.
	username := strings.TrimPrefix(value, "@")

	// RULE: An account that names a host must name THIS host. Otherwise "bob@other.example"
	// describes an account elsewhere, and treating it as a naked username would query the database
	// for a row that cannot exist -- usernames are letters, numbers, and underscores only
	// (service.User.ValidateUsername), so an embedded "@" always separates a host. RFC 7033
	// Section 4.5: this server answers only for the resources it is authoritative for. (BUG-18)
	if before, after, found := strings.Cut(username, "@"); found {

		if uri.Hostname(after) != hostname {
			return "", ""
		}

		username = before
	}

	// An empty username names nothing. Left to fall through, it would look up a User whose token is
	// the empty string, which is a database query with no possible answer.
	if username == "" {
		return "", ""
	}

	// Special case for "Application" account
	if username == "application" {
		return model.ActorTypeApplication, ""
	}

	// Special case for Global Search actor
	if username == "search" {
		return model.ActorTypeSearchDomain, ""
	}

	// Special case for SearchQuery objects
	if searchQueryID, found := strings.CutPrefix(username, "search_"); found {
		return model.ActorTypeSearchQuery, searchQueryID
	}

	// Otherwise, it's a User.  A naked username ("benpate") lands here too -- that leniency is
	// deliberate, and the host check above is careful to allow it.
	return model.ActorTypeUser, username
}
