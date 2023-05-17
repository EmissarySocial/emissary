package service

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Locator struct {
	userService   *User
	streamService *Stream
	host          string
}

func NewLocator(userService *User, streamService *Stream, host string) Locator {
	return Locator{
		userService:   userService,
		streamService: streamService,
		host:          host,
	}
}

// GetObjectFromURL parses a URL and verifies the existence of the referenced object.
func (service Locator) GetObjectFromURL(value string) (string, primitive.ObjectID, error) {

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
