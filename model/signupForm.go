package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type SignupForm struct {
	Title   string             `path:"title"   json:"title"`   // Title displayed on the signup page
	Message string             `path:"message" json:"message"` // Message displayed on the signup page
	GroupID primitive.ObjectID `path:"groupId" json:"groupId"` // Group to add new users to when completed
	Active  bool               `path:"active"  json:"active"`  // If true, then allow this signup page
}
