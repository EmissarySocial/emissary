package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported OutboxMessage to the new profile.
func (service *Outbox) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.OutboxMessage.Import"

	// Unmarshal the JSON document into a new OutboxMessage
	outboxMessage := model.NewOutboxMessage()
	if err := json.Unmarshal(document, &outboxMessage); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = outboxMessage.OutboxMessageID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original OutboxMessage into the new, local OutboxMessage
	outboxMessage.OutboxMessageID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &outboxMessage.ActorID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map ActorID", "ActorID: "+user.UserID.Hex()+", OutboxMessageID: "+outboxMessage.OutboxMessageID.Hex()))
	}

	// Save the OutboxMessage to the database
	if err := service.Save(session, &outboxMessage, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported OutboxMessage")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported OutboxMessage from the database
func (service *Outbox) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.OutboxMessage.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
