package common

import (
	"net/url"
	"strings"

	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams/vocab"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ParseItem finds the item URL, then splits it into its component parts.
func ParseItem(item vocab.Type) (userID primitive.ObjectID, itemType string, itemID primitive.ObjectID, err error) {
	return ParseURL(item.GetJSONLDId().GetIRI())
}

// ParseURL splits a URL into its component parts: userID, itemType, itemID.
func ParseURL(url *url.URL) (userID primitive.ObjectID, itemType string, itemID primitive.ObjectID, err error) {

	const location = "service.activitypub.Database.parseURL"

	// Remove required prefix
	prefix := "/.activitypub/"

	if !strings.HasPrefix(url.Path, prefix) {
		err = derp.NewBadRequestError(location, "Path must begin with /.activitypub/", url.String())
		return
	}

	// Split the path into a list of arguments
	path := strings.Split(url.Path, "/")

	if len(path) == 0 {
		err = derp.NewBadRequestError(location, "Path must contain at least one argument", url.String())
		return
	}

	// Get the UserID from the path
	userID, err = primitive.ObjectIDFromHex(path[0])

	if err != nil {
		err = derp.NewBadRequestError(location, "Invalid userID", url, err)
		return
	}

	if len(path) == 1 {
		return
	}

	// Get the ItemType from the path
	itemType = path[1]

	if len(path) == 2 {
		return
	}

	// Get the ItemID from the path
	itemID, err = primitive.ObjectIDFromHex(path[2])

	if err != nil {
		err = derp.NewBadRequestError(location, "Invalid itemID", url, err)
		return
	}

	// Nothin else to get
	return
}
