package activitypub

import "go.mongodb.org/mongo-driver/bson/primitive"

type StorageService interface {
	LoadItemByID(userID primitive.ObjectID, itemID primitive.ObjectID) (any, error)
	SaveObject(any) error
}
