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

func (service *Product) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Product) ExportDocument(session data.Session, userID primitive.ObjectID, productID primitive.ObjectID) (string, error) {

	const location = "service.Product.ExportDocument"

	// Load the Product
	product := model.NewProduct()
	if err := service.LoadByUserAndID(session, userID, productID, &product); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Product")
	}

	// Marshal the product as JSON
	result, err := json.Marshal(product)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Product", product)
	}

	// Success
	return string(result), nil
}
