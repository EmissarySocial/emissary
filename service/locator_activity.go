package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityURL generates a URL for an Activity based on the ActorType and ActorID
func (service *Locator) ActivityURL(actorType string, actorID primitive.ObjectID, activityID primitive.ObjectID) string {

	switch actorType {

	case model.ActorTypeApplication:
		return service.host + "/@application/pub/outbox/" + activityID.Hex()

	case model.ActorTypeSearchDomain:
		return service.host + "/@search/pub/outbox/" + activityID.Hex()

	case model.ActorTypeSearchQuery:
		return service.host + "/@search_" + actorID.Hex() + "/pub/outbox/" + activityID.Hex()

	case model.ActorTypeStream:
		return service.host + "/" + actorID.Hex() + "/pub/outbox/" + activityID.Hex()

	case model.ActorTypeUser:
		return service.host + "/@" + actorID.Hex() + "/pub/outbox/" + activityID.Hex()

	default:
		return ""
	}
}

// ParseActivity splits the URL of a User's or a Stream's activity on this server into its actor type,
// actor ID, and activity ID. It reads the URLs that ActivityURL writes for those two actor types.
func (service *Locator) ParseActivity(url string) (string, primitive.ObjectID, primitive.ObjectID, error) {

	const location = "service.Locator.ParseActivity"

	// RULE: Only a URL on this host names one of our activities
	path, found := strings.CutPrefix(url, service.host+"/")

	if !found {
		return "", primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "Unable to parse Activity URL", "url: "+url)
	}

	// A User's outbox is under "@" and their ID, and a Stream's is under its ID alone
	actorType := model.ActorTypeStream

	if userPath, isUser := strings.CutPrefix(path, "@"); isUser {
		actorType = model.ActorTypeUser
		path = userPath
	}

	// Isolate the actor token and activity token
	actorToken, activityToken, found := strings.Cut(path, "/pub/outbox/")

	if !found {
		return "", primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "Unable to parse Activity URL", "url: "+url)
	}

	// Parse the ActorID
	actorID, err := primitive.ObjectIDFromHex(actorToken)

	if err != nil {
		return "", primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "Invalid Actor ID", "actorToken: "+actorToken)
	}

	// Parse the ActivityID
	activityID, err := primitive.ObjectIDFromHex(activityToken)

	if err != nil {
		return "", primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "Invalid Activity ID", "activityToken: "+activityToken)
	}

	return actorType, actorID, activityID, nil
}
