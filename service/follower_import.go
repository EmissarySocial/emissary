package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Follower to the new profile.
func (service *Follower) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Follower.Import"

	// Unmarshal the JSON document into a new Follower
	follower := model.NewFollower()
	if err := json.Unmarshal(document, &follower); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = follower.FollowerID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Follower into the new, local Follower
	follower.FollowerID = importItem.LocalID            // Use the new localID for this record
	follower.StateID = model.FollowerStateImportPending // Set a new State ID to finalize when we `Move`

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &follower.ParentID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map ParentID", "ParentID: "+user.UserID.Hex()+", FollowerID: "+follower.FollowerID.Hex()))
	}

	// Save the Follower to the database
	if err := service.Save(session, &follower, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Follower")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Follower from the database
func (service *Follower) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Follower.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
