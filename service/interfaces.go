package service

import (
	"io"

	"github.com/benpate/data"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthorSetter interface {
	SetAuthor(object data.Object, authorID primitive.ObjectID) error
}

type TemplateLike interface {
	Execute(writer io.Writer, data any) error
}
