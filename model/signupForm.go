package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SignupForm struct {
	Title   string             `json:"title"   bson:"title"`   // Title displayed on the signup page
	Message string             `json:"message" bson:"message"` // Message displayed on the signup page
	GroupID primitive.ObjectID `json:"groupId" bson:"groupId"` // Group to add new users to when completed
	Active  bool               `json:"active"  bson:"active"`  // If true, then allow this signup page
}

func NewSignupForm() SignupForm {
	return SignupForm{}
}
