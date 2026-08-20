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

// ExportCollection returns the IDs of every Privilege to include in a User's data export
func (service *Privilege) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

// ExportDocument returns a single Privilege as a JSON string, for a User's data export
func (service *Privilege) ExportDocument(session data.Session, userID primitive.ObjectID, privilegeID primitive.ObjectID) (string, error) {

	const location = "service.Privilege.ExportDocument"

	// Load the Privilege
	privilege := model.NewPrivilege()
	if err := service.LoadByID(session, userID, privilegeID, &privilege); err != nil {
		return "", derp.Wrap(err, location, "Loading Privilege")
	}

	// Marshal the privilege as JSON
	result, err := json.Marshal(privilege)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshaling Privilege", privilege)
	}

	// Success
	return string(result), nil
}
