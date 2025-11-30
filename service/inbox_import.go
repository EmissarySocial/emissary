package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Inbox to the new profile.
func (service *Inbox) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Inbox.Import"

	// Unmarshal the JSON document into a new Inbox
	message := model.NewMessage()
	if err := json.Unmarshal(document, &message); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = message.MessageID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Inbox into the new, local Inbox
	message.MessageID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &message.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", InboxID: "+message.MessageID.Hex()))
	}

	// Map the FollowingID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &message.FollowingID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map FollowingID", "UserID: "+user.UserID.Hex()+", InboxID: "+message.MessageID.Hex()))
	}

	// Map the FolderID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &message.FolderID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map FolderID", "UserID: "+user.UserID.Hex()+", InboxID: "+message.MessageID.Hex()))
	}

	// Save the Inbox to the database
	if err := service.Save(session, &message, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Inbox")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Inbox from the database
func (service *Inbox) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Inbox.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
