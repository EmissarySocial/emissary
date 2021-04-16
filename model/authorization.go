package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Authorization struct {
	UserID   primitive.ObjectID
	GroupIDs []primitive.ObjectID
	IsOwner  bool
}
