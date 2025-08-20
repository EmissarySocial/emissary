package service

import (
	"crypto"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/hannibal/outbox"
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

func NewLocator() Locator {
	return Locator{}
}

func (service *Locator) Refresh(domainService *Domain, searchDomainService *SearchDomain, searchQueryService *SearchQuery, streamService *Stream, userService *User, host string) {

	service.domainService = domainService
	service.searchDomainService = searchDomainService
	service.streamService = streamService
	service.searchQueryService = searchQueryService
	service.userService = userService

	service.host = host
}

// GetWebFingerResult returns a digit.Resource object based on the provided resource string.
func (service *Locator) GetWebFingerResult(session data.Session, resource string) (digit.Resource, error) {

	const location = "service.Locator.GetWebFingerResult"

	objectType, token := locateObjectFromURL(service.host, resource)

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

	return digit.Resource{}, derp.BadRequestError(location, "Invalid Resource", resource)
}

// GetObjectFromURL parses a URL and verifies the existence of the referenced object.
func (service *Locator) GetObjectFromURL(session data.Session, value string) (string, primitive.ObjectID, error) {

	const location = "service.Locator.GetObjectFromURL"

	objectType, token := locateObjectFromURL(service.host, value)

	// Verify database records
	switch objectType {

	case model.ActorTypeApplication:
		return "", primitive.NilObjectID, derp.BadRequestError(location, "Invalid Object Type", objectType)

	case model.ActorTypeSearchDomain:
		return "", primitive.NilObjectID, derp.BadRequestError(location, "Invalid Object Type", objectType)

	case model.ActorTypeSearchQuery:
		return "", primitive.NilObjectID, derp.BadRequestError(location, "Invalid Object Type", objectType)

	case model.ActorTypeStream:

		stream := model.NewStream()

		if err := service.streamService.LoadByToken(session, token, &stream); err != nil {
			return "", primitive.NilObjectID, derp.Wrap(err, location, "Error loading stream", token)
		}

		return model.ActorTypeStream, stream.StreamID, nil

	case model.ActorTypeUser:

		user := model.NewUser()

		if err := service.userService.LoadByToken(session, token, &user); err != nil {
			return "", primitive.NilObjectID, derp.Wrap(err, location, "Error loading user", token)
		}

		return model.ActorTypeUser, user.UserID, nil
	}

	// Fall through is failure.  Feel bad.
	return "", primitive.NilObjectID, derp.BadRequestError(location, "Invalid Object Type", objectType)
}

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
		return outbox.Actor{}, derp.BadRequestError(location, "Invalid Actor Type", actorType)
	}

	return outbox.Actor{}, derp.BadRequestError(location, "ActorID must be a valid ObjectID", actorType)
}

func (service *Locator) GetPrivateKey(session data.Session, actorType string, actorID primitive.ObjectID) (publicKeyID string, privateKey crypto.PrivateKey, err error) {

	const location = "service.locator.GetPrivateKey"

	switch actorType {

	case model.ActorTypeApplication:
		publicKeyID := service.domainService.PublicKeyID()
		privateKey, err := service.domainService.PrivateKey(session)
		return publicKeyID, privateKey, err

	case model.ActorTypeSearchDomain:
		publicKeyID := service.domainService.PublicKeyID()
		privateKey, err := service.domainService.PrivateKey(session)
		return publicKeyID, privateKey, err

	case model.ActorTypeSearchQuery:
		publicKeyID := service.domainService.PublicKeyID()
		privateKey, err := service.domainService.PrivateKey(session)
		return publicKeyID, privateKey, err

	case model.ActorTypeStream:
		publicKeyID := service.streamService.PublicKeyID(actorID)
		privateKey, err := service.streamService.PrivateKey(session, actorID)
		return publicKeyID, privateKey, err

	case model.ActorTypeUser:
		publicKeyID := service.userService.PublicKeyID(actorID)
		privateKey, err := service.userService.PrivateKey(session, actorID)
		return publicKeyID, privateKey, err
	}

	return "", nil, derp.BadRequestError(location, "Invalid Actor Type", actorType)
}

// locateObjectFromURL parses a URL, determines what type of object it is,
// and extracts the objectID.  It requires the current host (protocol + domain)
// to match and the complete URL to be looked up. The returned object type
// can be one of: (Stream, User, SearchQuery, or Service).  If the object
// is not found, then both the type and token will be empty strings.
func locateObjectFromURL(host string, value string) (string, string) {

	hostname := domain.NameOnly(host)

	// Identify Username-type values
	if value, found := strings.CutSuffix(value, "@"+hostname); found {

		value = strings.TrimPrefix(value, "acct:")
		value = strings.TrimPrefix(value, "@")

		// Special case for "Application" account
		if value == "application" {
			return model.ActorTypeApplication, ""
		}

		// Special case for Global Search actor
		if value == "search" {
			return model.ActorTypeSearchDomain, ""
		}

		// Special case for SearchQuery objects
		if value, found := strings.CutPrefix(value, "search_"); found {
			return model.ActorTypeSearchQuery, value
		}

		// Otherwise, it's a User
		return model.ActorTypeUser, value
	}

	// Identify URL-type values
	if value, found := strings.CutPrefix(value, host); found {

		// Remove leading slash and query params (if present)
		value = strings.TrimPrefix(value, "/")
		value, _, _ = strings.Cut(value, "?")
		value, _, _ = strings.Cut(value, "/")

		// Special case for "Application" account
		if value == "" {
			return model.ActorTypeApplication, ""
		}
		// Special case for "Application" account
		if value == "@application" {
			return model.ActorTypeApplication, ""
		}

		// Identify Gloabl Search actor
		if value == "@search" {
			return model.ActorTypeSearchDomain, ""
		}

		// Identify SearchQuery URLs
		if value, found := strings.CutPrefix(value, "@search_"); found {
			return model.ActorTypeSearchQuery, value
		}

		// Identify User URLs
		if value, found := strings.CutPrefix(value, "@"); found {
			value, _, _ = strings.Cut(value, "/")
			return model.ActorTypeUser, value
		}

		// Trim off any trailing path data
		return model.ActorTypeStream, value
	}

	return "", ""
}
