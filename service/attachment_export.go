package service

import (
	"encoding/json"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *Attachment) ExportCollection(session data.Session, objectID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("objectType", model.AttachmentObjectTypeStream).AndEqual("objectId", objectID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *Attachment) ExportDocument(session data.Session, objectID primitive.ObjectID, attachmentID primitive.ObjectID) (string, error) {

	const location = "service.Attachment.ExportDocument"

	// Load the Attachment
	attachment := model.NewAttachment("", primitive.NilObjectID)
	if err := service.LoadByID(session, model.AttachmentObjectTypeStream, objectID, attachmentID, &attachment); err != nil {
		return "", derp.Wrap(err, location, "Unable to load Attachment")
	}

	// Marshal the attachment as JSON
	result, err := json.Marshal(attachment)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal Attachment", attachment)
	}

	// Success
	return string(result), nil
}

func (service *Attachment) ExportOriginal(session data.Session, objectType string, objectID primitive.ObjectID, attachmentID primitive.ObjectID, request *http.Request, writer http.ResponseWriter) error {

	const location = "service.Attachment.ExportOriginal"

	// Load the Attachment from the database
	attachment := model.NewAttachment(objectType, objectID)

	if err := service.LoadByID(session, objectType, objectID, attachmentID, &attachment); err != nil {
		return derp.Wrap(err, location, "Unable to load Attachment", attachmentID)
	}

	// Serve the original file via HTTP
	if err := service.mediaServer.ServeOriginal(writer, request, attachment.AttachmentID.Hex()); err != nil {
		return derp.Wrap(err, location, "Unable to serve original file", attachment.AttachmentID)
	}

	// Sucksess
	return nil
}
