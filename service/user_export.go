package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExportCollection returns the IDs of every User to include in a User's data export
func (service *User) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	return []model.IDOnly{
		{ID: userID},
	}, nil
}

// ExportDocument returns a single User as a JSON string, for a User's data export
func (service *User) ExportDocument(session data.Session, userID primitive.ObjectID, _ primitive.ObjectID) (string, error) {

	const location = "service.User.ExportDocument"

	// Load the User
	user := model.NewUser()
	if err := service.LoadByID(session, userID, &user); err != nil {
		return "", derp.Wrap(err, location, "Loading User")
	}

	// Marshal the user as JSON
	result, err := json.Marshal(user)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshaling User", user)
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
		return derp.Wrap(err, location, "Saving user")
	}

	// Background task (post-commit) to delete records and send `Move` notifications to followers.
	postcommit.Publish(session, service.queue, "MoveUser", mapof.Any{
		"hostname": service.Hostname(),
		"userId":   user.UserID.Hex(),
		"actor":    actor,
		"oracle":   oracle,
	})

	return nil
}
