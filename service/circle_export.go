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

// ExportCollection returns the IDs of every Circle to include in a User's data export
func (service *Circle) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

// ExportDocument returns a single Circle as a JSON string, for a User's data export
func (service *Circle) ExportDocument(session data.Session, userID primitive.ObjectID, circleID primitive.ObjectID) (string, error) {

	const location = "service.Circle.ExportDocument"

	// Load the Circle
	circle := model.NewCircle()
	if err := service.LoadByID(session, userID, circleID, &circle); err != nil {
		return "", derp.Wrap(err, location, "Loading Circle")
	}

	// Marshal the circle as JSON
	result, err := json.Marshal(circle)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshaling Circle", circle)
	}

	// Success
	return string(result), nil
}
