package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Conversation to the new profile.
func (service *Conversation) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Conversation.Import"

	// Unmarshal the JSON document into a new Conversation
	conversation := model.NewConversation()
	if err := json.Unmarshal(document, &conversation); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = conversation.ConversationID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Conversation into the new, local Conversation
	conversation.ConversationID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &conversation.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", ConversationID: "+conversation.ConversationID.Hex()))
	}

	// Save the Conversation to the database
	if err := service.Save(session, &conversation, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Conversation")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Conversation from the database
func (service *Conversation) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Conversation.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
