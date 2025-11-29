package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *User) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	return []model.IDOnly{
		{ID: userID},
	}, nil
}

func (service *User) ExportDocument(session data.Session, userID primitive.ObjectID, _ primitive.ObjectID) (string, error) {

	const location = "service.User.ExportDocument"

	// Load the User
	user := model.NewUser()
	if err := service.LoadByID(session, userID, &user); err != nil {
		return "", derp.Wrap(err, location, "Unable to load User")
	}

	// Marshal the user as JSON
	result, err := json.Marshal(user)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal User", user)
	}

	// Success
	return string(result), nil
}
