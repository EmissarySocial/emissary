package service

import (
	"github.com/benpate/data"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthorSetter interface {
	SetAuthor(object data.Object, authorID primitive.ObjectID) error
}
