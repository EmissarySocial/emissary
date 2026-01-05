package service

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

// This only works for Users at the moment
func (service *Locator) ParseActivity(url string) (string, primitive.ObjectID, primitive.ObjectID, error) {
	const location = "canonical.ParseActivity"

	if strings.HasPrefix(url, service.host+"/@") {

		// Isolate the actor token and activity token
		actorToken := strings.TrimPrefix(url, service.host+"/@")
		actorToken, activityToken, found := strings.Cut(actorToken, "/pub/outbox/")
		if !found {
			return "", primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "Unable to parse Activity URL", "url", url)
		}

		// Parse the ActorID
		actorID, err := primitive.ObjectIDFromHex(actorToken)
		if err != nil {
			return "", primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "Invalid Actor ID", "actorToken", actorToken)
		}

		// Parse the ActivityID
		activityID, err := primitive.ObjectIDFromHex(activityToken)
		if err != nil {
			return "", primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "Invalid Activity ID", "activityToken", activityToken)
		}

		return model.ActorTypeUser, actorID, activityID, nil
	}

	return "", primitive.NilObjectID, primitive.NilObjectID, derp.BadRequest(location, "Unable to parse Activity URL", "url", url)
}
