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

func (service *Circle) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Circle) ExportDocument(session data.Session, userID primitive.ObjectID, circleID primitive.ObjectID) (string, error) {

	const location = "service.Circle.ExportDocument"

	// Load the Circle
	circle := model.NewCircle()
	if err := service.LoadByID(session, userID, circleID, &circle); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Circle")
	}

	// Marshal the circle as JSON
	result, err := json.Marshal(circle)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Circle", circle)
	}

	// Success
	return string(result), nil
}
