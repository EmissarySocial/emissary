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

// ExportCollection returns the IDs of every CollectionItem to include in a User's data export
func (service *CollectionItem) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

// ExportDocument returns a single CollectionItem as a JSON string, for a User's data export
func (service *CollectionItem) ExportDocument(session data.Session, userID primitive.ObjectID, collectionItemID primitive.ObjectID) (string, error) {

	const location = "service.CollectionItem.ExportDocument"

	// Load the CollectionItem
	collectionItem := model.NewCollectionItem()
	if err := service.LoadByID(session, userID, collectionItemID, &collectionItem); err != nil {
		return "", derp.Wrap(err, location, "Loading CollectionItem")
	}

	// Marshal the collectionItem as JSON
	result, err := json.Marshal(collectionItem)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshaling CollectionItem", collectionItem)
	}

	// Success
	return string(result), nil
}
