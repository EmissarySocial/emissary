package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Locator is used to find objects based on their URL or WebFinger token
type Locator struct {
	domainService      *Domain
	searchQueryService *SearchQuery
	streamService      *Stream
	userService        *User
	host               string
}

func NewLocator() Locator {
	return Locator{}
}

func (service *Locator) Refresh(domainService *Domain, searchQueryService *SearchQuery, streamService *Stream, userService *User, host string) {
	service.domainService = domainService
	service.streamService = streamService
	service.searchQueryService = searchQueryService
	service.userService = userService
	service.host = host
}

// GetWebFingerResult returns a digit.Resource object based on the provided resource string.
func (service *Locator) GetWebFingerResult(resource string) (digit.Resource, error) {

	const location = "service.Locator.GetWebFingerResult"

	objectType, token := locateObjectFromURL(service.host, resource)

	switch objectType {

	case "Stream":
		return service.streamService.WebFinger(token)

	case "SearchQuery":
		return service.searchQueryService.WebFinger(token)

	case "User":
		return service.userService.WebFinger(token)

	case "Application":
		return service.domainService.WebFinger(), nil
	}

	return digit.Resource{}, derp.NewBadRequestError(location, "Invalid Resource", resource)
}

// GetObjectFromURL parses a URL and verifies the existence of the referenced object.
func (service *Locator) GetObjectFromURL(value string) (string, primitive.ObjectID, error) {

	const location = "service.Locator.GetObjectFromURL"

	objectType, token := locateObjectFromURL(service.host, value)

	// Verify database records
	switch objectType {

	case "User":

		user := model.NewUser()

		if err := service.userService.LoadByToken(token, &user); err != nil {
			return "", primitive.NilObjectID, derp.Wrap(err, location, "Error loading user", token)
		}

		return "User", user.UserID, nil

	case "Stream":

		stream := model.NewStream()

		if err := service.streamService.LoadByToken(token, &stream); err != nil {
			return "", primitive.NilObjectID, derp.Wrap(err, location, "Error loading stream", token)
		}

		return "Stream", stream.StreamID, nil

	}

	// Fall through is failure.  Feel bad.
	return "", primitive.NilObjectID, derp.NewBadRequestError(location, "Invalid Object Type", objectType)
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

		value = strings.TrimSuffix(value, "@"+hostname)
		value = strings.TrimPrefix(value, "acct:")
		value = strings.TrimPrefix(value, "@")

		// Special case for "Application" account
		if value == "application" {
			return "Application", ""
		}

		// Special case for SearchQuery objects
		if value, found := strings.CutPrefix(value, "search_"); found {
			return "SearchQuery", value
		}

		// Otherwise, it's a User
		return "User", value
	}

	// Identify URL-type values
	if value, found := strings.CutPrefix(value, host); found {

		// Remove leading slash and query params (if present)
		value = strings.TrimPrefix(value, "/")
		value, _, _ = strings.Cut(value, "?")
		value, _, _ = strings.Cut(value, "/")

		// Special case for "Application" account
		if value == "" {
			return "Application", ""
		}
		// Special case for "Application" account
		if value == "@application" {
			return "Application", ""
		}

		// Identify SearchQuery URLs
		if value, found := strings.CutPrefix(value, "@search_"); found {
			return "SearchQuery", value
		}

		// Identify User URLs
		if value, found := strings.CutPrefix(value, "@"); found {
			value, _, _ = strings.Cut(value, "/")
			return "User", value
		}

		// Trim off any trailing path data
		return "Stream", value
	}

	return "", ""
}
