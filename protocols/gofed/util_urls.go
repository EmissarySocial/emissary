package gofed

import (
	"net/url"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/go-fed/activity/streams/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// IsLocalURL returns TRUE if the provided URL is owned by this server.
func IsLocalURL(hostname string, id *url.URL) bool {

	if id == nil {
		return false
	}

	return strings.HasPrefix(id.String(), hostname)
}

// ParseItem finds the item URL, then splits it into its component parts.
func ParseItem(item vocab.Type) (userID primitive.ObjectID, activityLocation string, activityID primitive.ObjectID, err error) {
	return ParseURL(item.GetJSONLDId().GetIRI())
}

// ParseURL splits a URL into its component parts: userID, activityLocation, activityID.
func ParseURL(url *url.URL) (userID primitive.ObjectID, activityLocation string, activityID primitive.ObjectID, err error) {

	const location = "service.activitypub.Database.parseURL"

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
		return userID, "", primitive.NilObjectID, derp.Wrap(err, location, "Invalid userID", userIDstring)
	}

	if path.IsEmpty() {
		return userID, "", primitive.NilObjectID, nil
	}

	// Parse the activityLocation from the path
	activityLocation, path = path.Split()

	if path.IsEmpty() {
		return userID, activityLocation, primitive.NilObjectID, nil
	}

	// Parse the activityID from the path
	var activityIDstring string
	activityIDstring, _ = path.Split()

	activityID, err = primitive.ObjectIDFromHex(activityIDstring)

	if err != nil {
		return userID, activityLocation, primitive.NilObjectID, derp.Wrap(err, location, "Invalid activityID", activityIDstring)
	}

	// Success.  All values parsed correctly.
	return userID, activityLocation, activityID, nil
}
