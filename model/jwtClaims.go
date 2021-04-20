package model

import (
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JWTClaims struct {
	UserID   primitive.ObjectID   `json:"userId"`
	GroupIDs []primitive.ObjectID `json:"groupIds"`
	Owner    bool                 `json:"owner"`

	jwt.StandardClaims
}
