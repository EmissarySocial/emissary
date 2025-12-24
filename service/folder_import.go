package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Folder to the new profile.
func (service *Folder) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Folder.Import"

	// Unmarshal the JSON document into a new Folder
	folder := model.NewFolder()
	if err := json.Unmarshal(document, &folder); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update the ImportItem with the RemoteID
	importItem.RemoteID = folder.FolderID

	// If we already have a folder with the same label, then don't make a new record,
	// just map to the existing one
	existingFolder := model.NewFolder()
	if err := service.LoadByLabel(session, user.UserID, folder.Label, &existingFolder); err == nil {
		importItem.LocalID = existingFolder.FolderID
		return nil
	}

	// Fall through means we're going to create a new Folder.
	// Map values from the original Folder into the new, local Folder
	folder.FolderID = primitive.NewObjectID() // Use the new localID for this record
	importItem.LocalID = folder.FolderID      // Update the ImportItem with the new LocalID

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &folder.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", FolderID: "+folder.FolderID.Hex()))
	}

	// Save the Folder to the database
	if err := service.Save(session, &folder, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Folder")
	}

	// A Man, A Plan, A Canal. Paama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Folder from the database
func (service *Folder) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Folder.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
