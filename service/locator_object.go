package service

import (
	"strings"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ObjectURL returns the canonical URL for the specified object
func (service *Locator) ObjectURL(userID primitive.ObjectID, objectID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex() + "/pub/objects/" + objectID.Hex()
}

// ParseObject parses the userID and objectID from the specified URL
func (service *Locator) ParseObject(url string) (primitive.ObjectID, primitive.ObjectID, error) {
	const location = "canonical.ParseObject"

	// Verify that the URL looks correct (starts with https://host.social/@)
	if !strings.HasPrefix(url, service.host+"/@") {
		return primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "URL must match host")
	}

	// Isolate the user token and object token
	url = strings.TrimPrefix(url, service.host+"/@")
	userToken, objectToken, found := strings.Cut(url, "/pub/objects/")

	if !found {
		return primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "URL is not a valid Object URL")
	}

	// Parse the UserID
	userID, err := primitive.ObjectIDFromHex(userToken)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, location, "Invalid user ID in URL", "userToken", userToken)
	}

	// Parse the ObjectID
	objectID, err := primitive.ObjectIDFromHex(objectToken)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, derp.Wrap(err, location, "Invalid object ID in URL", "objectToken", objectToken)
	}

	// Success
	return userID, objectID, nil
}
