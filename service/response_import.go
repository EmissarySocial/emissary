package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Response to the new profile.
func (service *Response) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Response.Import"

	// Unmarshal the JSON document into a new Response
	response := model.NewResponse()
	if err := json.Unmarshal(document, &response); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = response.ResponseID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Response into the new, local Response
	response.ResponseID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &response.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", ResponseID: "+response.ResponseID.Hex()))
	}

	// Save the Response to the database
	if err := service.Save(session, &response, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Response")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Response from the database
func (service *Response) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Response.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
