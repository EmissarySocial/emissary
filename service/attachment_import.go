package service

import (
	"bytes"
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Attachment to the new profile.
func (service *Attachment) Import(session data.Session, record *model.Import, importItem *model.ImportItem, objectID primitive.ObjectID, document []byte) (remoteID primitive.ObjectID, remoteURL string, localID primitive.ObjectID, localURL string, err error) {

	const location = "service.Attachment.Import"

	// Unmarshal the JSON document into a new Attachment
	attachment := model.NewAttachment("", primitive.NilObjectID)
	if err := json.Unmarshal(document, &attachment); err != nil {
		return primitive.NilObjectID, "",
			primitive.NilObjectID, "",
			derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Get mapping IDs
	remoteID = attachment.AttachmentID
	remoteURL = attachment.URL
	localID = primitive.NewObjectID()
	localURL = "" // to be calculated below

	// Get the original file over HTTP
	var buffer bytes.Buffer
	txn := remote.Get(attachment.URL).
		With(options.BearerAuth(record.OAuthToken.AccessToken)).
		Result(&buffer)

	if err := txn.Send(); err != nil {
		return primitive.NilObjectID, "",
			primitive.NilObjectID, "",
			derp.Wrap(err, location, "Unable to retrieve original attachment file")
	}

	// Save the original file to the mediaserver
	if err := service.mediaServer.Put(localID.Hex(), &buffer); err != nil {
		return primitive.NilObjectID, "",
			primitive.NilObjectID, "",
			derp.Wrap(err, location, "Umable to save original document")
	}

	// Map values from the original Attachment into the new, local Attachment
	attachment.ObjectID = objectID    // Use the new parent ID for this record
	attachment.AttachmentID = localID // Use the new localID for this record
	attachment.URL = ""               // This will be recalculated on save

	// Save the Attachment to the database
	if err := service.Save(session, &attachment, "Imported"); err != nil {
		return primitive.NilObjectID, "",
			primitive.NilObjectID, "",
			derp.Wrap(err, location, "Unable to save imported Attachment")
	}

	localURL = attachment.URL

	// A Man, A Plan, A Canal. Panama.
	return remoteID, remoteURL, localID, localURL, nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Attachment from the database
func (service *Attachment) UndoImport(session data.Session, userID primitive.ObjectID, attachmentID primitive.ObjectID) error {

	const location = "service.Attachment.UndoImport"

	if err := service.HardDeleteByID(session, userID, attachmentID); err != nil {
		return derp.Wrap(err, location, "Unable to delete attachment", attachmentID)
	}

	// Delete uploaded files from MediaServer
	if err := service.mediaServer.Delete(attachmentID.Hex()); err != nil {
		derp.Report(derp.Wrap(err, "service.Attachment", "Error deleting attached files", attachmentID))
		// Fail loudly, but do not stop.
	}

	return nil
}
