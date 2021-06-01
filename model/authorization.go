package model

import (
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorization represents the JWT Claims that the server gives to a user when they sign in.
type Authorization struct {
	UserID      primitive.ObjectID   `json:"U"` // Unique identifier of the User
	GroupIDs    []primitive.ObjectID `json:"G"` // IDs for all server-level groups that the User belongs to
	DomainOwner bool                 `json:"D"` // If TRUE, then this user is an owner of this domain

	jwt.StandardClaims // By embedding the "StandardClaims" object, this record can support standard behaviors, like token expieration, etc.
}
