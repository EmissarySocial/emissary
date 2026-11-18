package service

import (
	"github.com/EmissarySocial/emissary/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActorURL returns the canonical URL of a User or Stream actor, and an empty string for any other actor type
func (service *Locator) ActorURL(actorType string, actorID primitive.ObjectID) string {

	switch actorType {

	case model.ActorTypeUser:
		return service.UserURL(actorID)

	case model.ActorTypeStream:
		return service.host + "/" + actorID.Hex()

	default:
		return ""
	}
}

// FollowersURL returns the URL of a User or Stream actor's followers collection, and an empty
// string for any other actor type
func (service *Locator) FollowersURL(actorType string, actorID primitive.ObjectID) string {

	actorURL := service.ActorURL(actorType, actorID)

	if actorURL == "" {
		return ""
	}

	return actorURL + "/pub/followers"
}
