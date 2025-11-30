package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Following to the new profile.
func (service *Following) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Following.Import"

	// Unmarshal the JSON document into a new Following
	following := model.NewFollowing()
	if err := json.Unmarshal(document, &following); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = following.FollowingID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Following into the new, local Following
	following.FollowingID = importItem.LocalID            // Use the new localID for this record
	following.Status = model.FollowingStatusImportPending // Wait until the migration is finalized to "Follow" these remote accounts.

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &following.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", FollowingID: "+following.FollowingID.Hex()))
	}

	// Map the FolderID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &following.FolderID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map FolderID", "UserID: "+user.UserID.Hex()+", FollowingID: "+following.FollowingID.Hex()))
	}

	// Save the Following to the database
	if err := service.Save(session, &following, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Following")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Following from the database
func (service *Following) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Following.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
