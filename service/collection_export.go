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

// ExportCollection returns the IDs of every Collection to include in a User's data export
func (service *Collection) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

// ExportDocument returns a single Collection as a JSON string, for a User's data export
func (service *Collection) ExportDocument(session data.Session, userID primitive.ObjectID, collectionID primitive.ObjectID) (string, error) {

	const location = "service.Collection.ExportDocument"

	// Load the Collection
	collection := model.NewCollection()
	if err := service.LoadByID(session, userID, collectionID, &collection); err != nil {
		return "", derp.Wrap(err, location, "Loading Collection")
	}

	// Marshal the collection as JSON
	result, err := json.Marshal(collection)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshaling Collection", collection)
	}

	// Success
	return string(result), nil
}
