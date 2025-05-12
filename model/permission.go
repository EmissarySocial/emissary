package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Permission struct {
	AllowAnonymous     bool
	AllowAuthenticated bool
	AllowMyself        bool
	AllowAuthor        bool
	AllowGroups        []primitive.ObjectID
	AllowProducts      []primitive.ObjectID
}
