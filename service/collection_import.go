package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Collection to the new profile.
func (service *Collection) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Collection.Import"

	// Unmarshal the JSON document into a new Collection
	collection := model.NewCollection()
	if err := json.Unmarshal(document, &collection); err != nil {
		return derp.Wrap(err, location, "Parsing remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = collection.CollectionID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Collection into the new, local Collection
	collection.CollectionID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &collection.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Mapping UserID", "UserID: "+user.UserID.Hex()+", CollectionID: "+collection.CollectionID.Hex()))
	}

	// Save the Collection to the database
	if err := service.Save(session, &collection, "Imported"); err != nil {
		return derp.Wrap(err, location, "Saving imported Collection")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Collection from the database
func (service *Collection) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Collection.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Deleting record", importItem.LocalID)
	}

	return nil
}
