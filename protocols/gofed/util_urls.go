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
	return parsePathWithLocation(url, model.ActivityPlaceInbox)
}

// ParseOutboxPath parses the path parameters in a URL and ensures that it identifies a user's outbox
func ParseOutboxPath(url *url.URL) (userID primitive.ObjectID, activityID primitive.ObjectID, err error) {
	return parsePathWithLocation(url, model.ActivityPlaceOutbox)
}

// parsePathWithLocation parses the path parameters in a URL and ensures that it itentifies a specific kind of record
func parsePathWithLocation(url *url.URL, expectedActivityPlace model.ActivityPlace) (primitive.ObjectID, primitive.ObjectID, error) {

	ownerID, activityPlace, itemID, err := ParsePath(url)

	if err != nil {
		return ownerID, itemID, err
	}

	if activityPlace != expectedActivityPlace {
		err = derp.NewBadRequestError("service.activitypub.ParseWithLocation", "Expected location is not correct", url.String(), expectedActivityPlace.String())
	}

	return ownerID, itemID, err
}

// ParsePath splits a URL into its component parts: userID, activityPlace, activityID.
func ParsePath(url *url.URL) (userID primitive.ObjectID, activityPlace model.ActivityPlace, activityID primitive.ObjectID, err error) {

	const location = "service.activitypub.Database.ParsePath"

	if !strings.HasPrefix(url.Path, "/@") {
		err = derp.NewBadRequestError(location, "Path must begin with /@", url.String())
	}

	path := list.BySlash(strings.TrimPrefix(url.Path, "/@"))

	if path.IsEmpty() {
		err = derp.NewBadRequestError(location, "Path is empty", url.String())
	}

	// Parse the UserID from the path
	var userIDstring string
	userIDstring, path = path.Split()

	userID, err = primitive.ObjectIDFromHex(userIDstring)

	if err != nil {
		return userID, model.ActivityPlaceUndefined, primitive.NilObjectID, derp.Wrap(err, location, "Invalid userID", userIDstring)
	}

	if path.IsEmpty() {
		return userID, model.ActivityPlaceUndefined, primitive.NilObjectID, nil
	}

	// Parse the activityPlace from the path
	var activityPlaceString string
	activityPlaceString, path = path.Split()

	activityPlace = model.ParseActivityPlace(activityPlaceString)

	if path.IsEmpty() {
		return userID, activityPlace, primitive.NilObjectID, nil
	}

	// Parse the activityID from the path
	var activityIDstring string
	activityIDstring, _ = path.Split()

	activityID, err = primitive.ObjectIDFromHex(activityIDstring)

	if err != nil {
		return userID, activityPlace, primitive.NilObjectID, derp.Wrap(err, location, "Invalid activityID", activityIDstring)
	}

	// Success.  All values parsed correctly.
	return userID, activityPlace, activityID, nil
}
