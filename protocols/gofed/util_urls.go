package gofed

import (
	"net/url"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
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

	var userIDstring string
	var itemIDstring string

	if !strings.HasPrefix(url.Path, "/@") {
		err = derp.NewBadRequestError(location, "Path must begin with /@", url.String())
	}

	path := list.BySlash(strings.TrimPrefix(url.Path, "/@"))

	if path.IsEmpty() {
		err = derp.NewBadRequestError(location, "Path is empty", url.String())
	}

	// Parse the UserID from the path
	userIDstring, path = path.Split()

	userID, err = primitive.ObjectIDFromHex(userIDstring)

	if err != nil {
		return userID, "", primitive.NilObjectID, derp.Wrap(err, location, "Invalid userID", userIDstring)
	}

	if path.IsEmpty() {
		return userID, "", primitive.NilObjectID, nil
	}

	// Parse the itemType from the path
	itemType, path = path.Split()

	if path.IsEmpty() {
		return userID, itemType, primitive.NilObjectID, nil
	}

	// Parse the itemID from the path
	itemIDstring, path = path.Split()

	itemID, err = primitive.ObjectIDFromHex(itemIDstring)

	if err != nil {
		return userID, itemType, primitive.NilObjectID, derp.Wrap(err, location, "Invalid itemID", itemIDstring)
	}

	// Success.  All values parsed correctly.
	return userID, itemType, itemID, nil
}
