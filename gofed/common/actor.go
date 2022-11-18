package common

import "go.mongodb.org/mongo-driver/bson/primitive"

func ActorURL(host string, userID primitive.ObjectID) string {
	return host + "/.activitypub/user" + userID.Hex()
}

func ActorInboxURL(host string, userID primitive.ObjectID) string {
	return host + "/.activitypub/user/" + userID.Hex() + "/inbox"
}

func ActorOutboxURL(host string, userID primitive.ObjectID) string {
	return host + "/.activitypub/user/" + userID.Hex() + "/outbox"
}
