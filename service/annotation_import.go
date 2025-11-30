package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Annotation to the new profile.
func (service *Annotation) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Annotation.Import"

	// Unmarshal the JSON document into a new Annotation
	annotation := model.NewAnnotation()
	if err := json.Unmarshal(document, &annotation); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = annotation.AnnotationID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Annotation into the new, local Annotation
	annotation.AnnotationID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &annotation.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", AnnotationID: "+annotation.AnnotationID.Hex()))
	}

	// Save the Annotation to the database
	if err := service.Save(session, &annotation, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Annotation")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Annotation from the database
func (service *Annotation) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Annotation.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
