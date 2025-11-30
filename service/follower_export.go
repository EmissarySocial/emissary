package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *Follower) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Follower) ExportDocument(session data.Session, userID primitive.ObjectID, followerID primitive.ObjectID) (string, error) {

	const location = "service.Follower.ExportDocument"

	// Load the Follower
	follower := model.NewFollower()
	if err := service.LoadByID(session, userID, followerID, &follower); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Follower")
	}

	// Marshal the follower as JSON
	result, err := json.Marshal(follower)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Follower", follower)
	}

	// Success
	return string(result), nil
}
