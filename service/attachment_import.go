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
			derp.Wrap(err, location, "Parsing remote document", document)
	}

	// Get mapping IDs
	remoteID = attachment.AttachmentID
	remoteURL = attachment.URL

	attachment.AttachmentID = primitive.NewObjectID()
	attachment.ObjectID = objectID // Use the new parent ID for this record
	localID = attachment.AttachmentID
	// localURL is calculated below

	originalURL := importItem.ImportURL + "/attachments/" + remoteID.Hex() + "/original"

	// Get the original file over HTTP
	var buffer bytes.Buffer
	txn := remote.Get(originalURL).
		With(options.BearerAuth(record.OAuthToken.AccessToken)).
		Result(&buffer)

	if err := txn.Send(); err != nil {
		return primitive.NilObjectID, "",
			primitive.NilObjectID, "",
			derp.Wrap(err, location, "Retrieving original attachment file")
	}

	// RULE: Sniff the imported file's own bytes to determine its content-type.
	// The remote server wrote both this file and the JSON that describes it, so
	// neither its "contentType" nor its "original" filename is taken at face value.
	// The filename only refines audio-vs-video inside a byte-confirmed container.
	attachment.ContentType = model.DetectContentType(buffer.Bytes(), attachment.Original)

	// Save the original file to the mediaserver
	if err := service.mediaServer.Put(localID.Hex(), &buffer); err != nil {
		return primitive.NilObjectID, "",
			primitive.NilObjectID, "",
			derp.Wrap(err, location, "Umable to save original document")
	}

	// Save the Attachment to the database
	if err := service.Save(session, &attachment, "Imported"); err != nil {
		return primitive.NilObjectID, "",
			primitive.NilObjectID, "",
			derp.Wrap(err, location, "Saving imported Attachment")
	}

	localURL = attachment.URL

	// A Man, A Plan, A Canal. Panama.
	return remoteID, remoteURL, localID, localURL, nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Attachment from the database
func (service *Attachment) UndoImport(session data.Session, userID primitive.ObjectID, attachmentID primitive.ObjectID) error {

	const location = "service.Attachment.UndoImport"

	if err := service.HardDeleteByID(session, userID, attachmentID); err != nil {
		return derp.Wrap(err, location, "Deleting attachment", attachmentID)
	}

	// Delete uploaded files from MediaServer
	if err := service.mediaServer.Delete(attachmentID.Hex()); err != nil {
		derp.Report(derp.Wrap(err, "service.Attachment", "Deleting attached files", attachmentID))
		// Fail loudly, but do not stop.
	}

	return nil
}
