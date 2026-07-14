package service

import (
	"strings"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionURL returns the canonical URL for the specified user
func (service *Locator) CollectionURL(userID primitive.ObjectID, collectionID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex() + "/pub/collections/" + collectionID.Hex()
}

// ParseCollection parses the userID and collectionID from the specified URL
func (service *Locator) ParseCollection(url string) (primitive.ObjectID, primitive.ObjectID, error) {

	const location = "service.Locator.ParseCollection"

	// Verify that the URL looks correct
	if !strings.HasPrefix(url, service.host+"/@") {
		return primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "URL must match host", "url: "+url, "host: "+service.host)
	}

	// Remove query string (if present)
	path, _, _ := strings.Cut(url, "?")

	// Isolate the user token and convert to ObjectID
	path = strings.TrimPrefix(path, service.host+"/@")
	userToken, collectionToken, found := strings.Cut(path, "/pub/collections/")

	if !found {
		return primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "URL must contain /pub/collections/", "url: "+url)
	}

	// Parse userID
	userID, err := primitive.ObjectIDFromHex(userToken)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, location, "Invalid user ID in URL", "url: "+url)
	}

	// Parse collectionID
	collectionID, err := primitive.ObjectIDFromHex(collectionToken)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, location, "Invalid collection ID in URL", "url: "+url)
	}

	// Success
	return userID, collectionID, nil
}
