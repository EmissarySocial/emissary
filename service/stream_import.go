package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Stream to the new profile.
func (service *Stream) Import(session data.Session, importRecord *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Stream.Import"

	// Unmarshal the JSON document into a new Stream
	stream := model.NewStream()
	if err := json.Unmarshal(document, &stream); err != nil {
		return derp.Wrap(err, location, "Parsing remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = stream.StreamID
	importItem.LocalID = primitive.NewObjectID()
	importItem.RemoteURL = stream.URL

	// Map values from the original Stream into the new, local Stream
	stream.StreamID = importItem.LocalID    // Use the new localID for this record
	stream.AttributedTo = user.PersonLink() // Associate the Stream with the LOCAL user
	stream.URL = ""                         // This will be recalculated by the StreamService.Save
	stream.CreateDate = 0                   // Reset the createDate so that we will INSERT the record

	// RULE: Every Stream must have a Template, but imported documents may not name one.
	// Default to "outbox-message", matching the Mastodon API (handler/mastodon/status.go).
	// This will be updated when we make a registry of default templates in profiles.
	if stream.TemplateID == "" {
		stream.TemplateID = "outbox-message"
	}

	// Map the ParentID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &stream.ParentID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Mapping ParentID. UserID: "+user.UserID.Hex()+", ParentID: "+stream.ParentID.Hex()))
	}

	// Map ParentIDs
	for index, parentID := range stream.ParentIDs {
		if err := service.importItemService.mapRemoteID(session, user.UserID, &stream.ParentIDs[index]); err != nil {
			return derp.Wrap(err, location, "Mapping ParentIDs", "index: "+convert.String(index), "ParentID: "+parentID.Hex())
		}
	}

	// Import and Map Attachments
	if err := service.importService.ImportAttachments(session, importRecord, importItem, &stream); err != nil {
		return derp.Wrap(err, location, "Importing Attachments")
	}

	// Save the Stream to the database
	if err := service.Save(session, &stream, "Imported"); err != nil {
		return derp.Wrap(err, location, "Saving imported Stream")
	}

	// The stream.URL is recalculated in the service.Save method.
	// Set the new (local) URL to the freshly-calculated URL for the new Stream record.
	importItem.LocalURL = stream.URL

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Stream from the database
func (service *Stream) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Stream.UndoImport"

	// Remove all Attachments
	attachments, err := service.attachmentService.QueryByObjectID(session, model.AttachmentObjectTypeStream, importItem.LocalID)

	if err != nil {
		return derp.Wrap(err, location, "Listing Attachments for Stream", importItem.LocalID)
	}

	for _, attachment := range attachments {
		if err := service.attachmentService.UndoImport(session, importItem.UserID, attachment.AttachmentID); err != nil {
			derp.Report(derp.Wrap(err, location, "Deleting Stream", attachment.AttachmentID))
		}
	}

	// Remove the Stream
	if err := service.HardDeleteByID(session, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Deleting Stream", importItem.LocalID)
	}

	return nil
}
