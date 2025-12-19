package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
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

// Move marks a User as "Moved" to the new actor location.  All requests from this User
// after this point should be rejected
func (service *User) Move(session data.Session, user *model.User, actor string, oracle string) error {

	const location = "service.User.Move"

	if actor == "" {
		return derp.BadRequest(location, "New actor URL must not be empty")
	}

	user.MovedTo = actor

	if err := service.Save(session, user, "Moved"); err != nil {
		return derp.Wrap(err, location, "Unable to save user")
	}

	// Background task to delete records and send `Move` notifications to followers.
	service.queue.NewTask("MoveUser", mapof.Any{
		"host":   service.Hostname(),
		"userId": user.UserID.Hex(),
		"actor":  actor,
		"oracle": oracle,
	})

	return nil
}
