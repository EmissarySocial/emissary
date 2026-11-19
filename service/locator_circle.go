package service

import (
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CircleURL returns the URL that names one of a User's circles as an audience
func (service *Locator) CircleURL(userID primitive.ObjectID, circleID primitive.ObjectID) string {
	return service.UserURL(userID) + "/pub/circles/" + circleID.Hex()
}

// ParseCircle splits a circle URL on this server into the User who owns the circle and the
// circle's ID. It returns FALSE for any other URL.
func (service *Locator) ParseCircle(url string) (primitive.ObjectID, primitive.ObjectID, bool) {

	// RULE: Only a URL on this host names one of our circles. A remote server that sends a URL
	// shaped like ours gains nothing.
	path, found := strings.CutPrefix(url, service.host+"/@")

	if !found {
		return primitive.NilObjectID, primitive.NilObjectID, false
	}

	userToken, circleToken, found := strings.Cut(path, "/pub/circles/")

	if !found {
		return primitive.NilObjectID, primitive.NilObjectID, false
	}

	userID, err := primitive.ObjectIDFromHex(userToken)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, false
	}

	circleID, err := primitive.ObjectIDFromHex(circleToken)

	if err != nil {
		return primitive.NilObjectID, primitive.NilObjectID, false
	}

	return userID, circleID, true
}
