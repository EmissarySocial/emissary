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

func (service *Following) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Following) ExportDocument(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID) (string, error) {

	const location = "service.Following.ExportDocument"

	// Load the Following
	following := model.NewFollowing()
	if err := service.LoadByID(session, userID, followingID, &following); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Following")
	}

	// Marshal the following as JSON
	result, err := json.Marshal(following)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Following", following)
	}

	// Success
	return string(result), nil
}
