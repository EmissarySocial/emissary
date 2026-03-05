package service

import (
	"strings"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserURL returns the canonical URL for the specified user
func (service *Locator) UserURL(host string, userID primitive.ObjectID) string {
	return service.host + "/@" + userID.Hex()
}

// ParseUser parses the userID from the specified URL
func (service *Locator) ParseUser(url string) (primitive.ObjectID, error) {

	const location = "canonical.ParseUser"

	// Verify that the URL looks correct
	if !strings.HasPrefix(url, service.host+"/@") {
		return primitive.NilObjectID, derp.BadRequest(location, "URL must match host")
	}

	// Isolate the user token and convert to ObjectID
	userToken := strings.TrimPrefix(url, service.host+"/@")
	userID, err := primitive.ObjectIDFromHex(userToken)

	if err != nil {
		return primitive.NilObjectID, derp.Wrap(err, location, "Invalid user ID in URL", "userToken", userToken)
	}

	// Success
	return userID, nil
}
