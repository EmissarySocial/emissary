package gofed

import (
	"net/url"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// IsLocalURL returns TRUE if the provided URL is owned by this server.
func IsLocalURL(hostname string, id *url.URL) bool {

	if id == nil {
		return false
	}

	return strings.HasPrefix(id.String(), hostname)
}

// ParseInboxPath parses the path parameters in a URL and ensures that it identifies a user's inbox
func ParseInboxPath(url *url.URL) (userID primitive.ObjectID, activityID primitive.ObjectID, err error) {
	return parsePathWithLocation(url, model.ActivityStreamContainerInbox)
}

// ParseOutboxPath parses the path parameters in a URL and ensures that it identifies a user's outbox
func ParseOutboxPath(url *url.URL) (userID primitive.ObjectID, activityID primitive.ObjectID, err error) {
	return parsePathWithLocation(url, model.ActivityStreamContainerOutbox)
}

// parsePathWithLocation parses the path parameters in a URL and ensures that it itentifies a specific kind of record
func parsePathWithLocation(url *url.URL, expectedActivityPlace model.ActivityStreamContainer) (primitive.ObjectID, primitive.ObjectID, error) {

	ownerID, container, itemID, err := ParsePath(url)

	if err != nil {
		return ownerID, itemID, err
	}

	if container != expectedActivityPlace {
		err = derp.NewBadRequestError("service.activitypub.ParseWithLocation", "Expected location is not correct", url.String(), expectedActivityPlace.String())
	}

	return ownerID, itemID, err
}

// ParsePath splits a URL into its component parts: userID, container, activityID.
func ParsePath(url *url.URL) (userID primitive.ObjectID, container model.ActivityStreamContainer, activityID primitive.ObjectID, err error) {

	const location = "service.activitypub.Database.ParsePath"

	userID = primitive.NilObjectID
	container = model.ActivityStreamContainerUndefined
	activityID = primitive.NilObjectID

	// Split the URL Path into a list.List
	if !strings.HasPrefix(url.Path, "/@") {
		return userID, container, activityID, derp.NewBadRequestError(location, "URL must be a recognizable ActivityPub path.", url.String())
	}

	path := list.BySlash(strings.TrimPrefix(url.Path, "/@"))

	if path.IsEmpty() {
		err = derp.NewBadRequestError(location, "Path must not be empty", url.String())
	}

	// Parse the UserID from the path
	var userIDstring string
	userIDstring, path = path.Split()
	userID, err = primitive.ObjectIDFromHex(userIDstring)

	if err != nil {
		return userID, container, activityID, derp.Wrap(err, location, "Invalid userID", userIDstring)
	}

	if path.IsEmpty() {
		return userID, container, activityID, nil
	}

	// The next item in the list MUST be /pub/
	if head := path.Head(); head != "pub" {
		return userID, container, activityID, derp.NewBadRequestError(location, "Path must begin with /@:userID/pub/", url.String())
	}

	path = path.Tail()

	// Parse the container from the path
	var containerString string
	containerString, path = path.Split()
	container = model.ParseActivityStreamContainer(containerString)

	if path.IsEmpty() {
		return userID, container, primitive.NilObjectID, nil
	}

	// Parse the activityID from the path
	activityIDstring := path.Head()
	activityID, err = primitive.ObjectIDFromHex(activityIDstring)

	if err != nil {
		return userID, container, activityID, derp.Wrap(err, location, "Invalid activityID", activityIDstring)
	}

	// Success.  All values parsed correctly.
	return userID, container, activityID, nil
}
