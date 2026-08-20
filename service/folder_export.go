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

// ExportCollection returns the IDs of every Folder to include in a User's data export
func (service *Folder) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

// ExportDocument returns a single Folder as a JSON string, for a User's data export
func (service *Folder) ExportDocument(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) (string, error) {

	const location = "service.Folder.ExportDocument"

	// Load the Folder
	folder := model.NewFolder()
	if err := service.LoadByID(session, userID, folderID, &folder); err != nil {
		return "", derp.Wrap(err, location, "Loading Folder")
	}

	// Marshal the folder as JSON
	result, err := json.Marshal(folder)

	if err != nil {
		return "", derp.Wrap(err, location, "Marshaling Folder", folder)
	}

	// Success
	return string(result), nil
}
