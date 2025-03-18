package service

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/domain"
	"github.com/benpate/rosetta/list"
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

	case "Service":
		return service.domainService.WebFinger(), nil
	}

	return digit.Resource{}, derp.NewBadRequestError(location, "Invalid Resource", resource)
}

// GetObjectFromURL parses a URL and verifies the existence of the referenced object.
func (service *Locator) GetObjectFromURL(value string) (string, primitive.ObjectID, error) {

	const location = "service.Locator.GetObjectFromURL"

	// Parse and validate the URL
	parsedURL, err := url.Parse(value)

	if err != nil {
		return "", primitive.NilObjectID, derp.Wrap(err, location, "Invalid URL", value)
	}

	if parsedURL.Host != service.host {
		return "", primitive.NilObjectID, derp.NewBadRequestError(location, "Invalid Host", parsedURL.Host)
	}

	// Look up the object/type
	class, token := getObjectFromURL(parsedURL)

	if token == "" {
		return "", primitive.NilObjectID, derp.Wrap(err, location, "Invalid URL", value)
	}

	// Verify database records
	switch class {
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
	return "", primitive.NilObjectID, derp.NewBadRequestError(location, "Invalid Object Type", class)
}

// getObjectFromURL parses a URL, determines what kind of object it is, and extracts the objectID
func getObjectFromURL(value *url.URL) (string, string) {

	path := strings.TrimPrefix(value.Path, "/")
	token := list.Slash(path).First()

	// Empty token the default page
	if token == "" {
		return "Stream", "home"
	}

	// Token starting with "@" is a user
	if strings.HasPrefix(token, "@") {
		return "User", strings.TrimPrefix(token, "@")
	}

	// Otherwise, it's a stream
	return "Stream", token
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

		// Special case for "service" account
		if value == "service" {
			return "Service", ""
		}

		// Special case for SearchQuery objects
		if value, found := strings.CutPrefix(value, "searchQuery_"); found {
			return "SearchQuery", value
		}

		// Otherwise, it's a User
		return "User", value
	}

	// Identify URL-type values
	if value, found := strings.CutPrefix(value, host); found {

		// Remove leading slash (if present)
		value = strings.TrimPrefix(value, "/")

		// Remove query parameters (if present)
		value, _, _ = strings.Cut(value, "?")

		// Special case for "Service" account
		if value == "@service" {
			return "Service", ""
		}

		// Special case for "Service" account
		if value == "" {
			return "Service", ""
		}

		// Identify SearchQuery URLs
		if value, found := strings.CutPrefix(value, ".search/"); found {
			value, _, _ = strings.Cut(value, "/")
			return "SearchQuery", value
		}

		// Identify User URLs
		if value, found := strings.CutPrefix(value, "@"); found {
			value, _, _ = strings.Cut(value, "/")
			return "User", value
		}

		// Trim off any trailing path data
		value, _, _ = strings.Cut(value, "/")
		return "Stream", value
	}

	return "", ""
}
