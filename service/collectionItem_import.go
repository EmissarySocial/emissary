package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported CollectionItem to the new profile.
func (service *CollectionItem) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.CollectionItem.Import"

	// Unmarshal the JSON document into a new CollectionItem
	collectionItem := model.NewCollectionItem()
	if err := json.Unmarshal(document, &collectionItem); err != nil {
		return derp.Wrap(err, location, "Parsing remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = collectionItem.CollectionItemID
	importItem.LocalID = primitive.NewObjectID()

	// Assign the new localID to this record's own primary key
	collectionItem.CollectionItemID = importItem.LocalID

	// Map the UserID from the remote value to its new local value
	if err := service.importItemService.mapRemoteID(session, user.UserID, &collectionItem.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Mapping UserID", "UserID: "+user.UserID.Hex()+", CollectionItemID: "+collectionItem.CollectionItemID.Hex()))
	}

	// Map the parent CollectionID from the remote value to its new local value
	if err := service.importItemService.mapRemoteID(session, user.UserID, &collectionItem.CollectionID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Mapping CollectionID", "UserID: "+user.UserID.Hex()+", CollectionID: "+collectionItem.CollectionID.Hex()))
	}

	// Save the CollectionItem to the database
	if err := service.Save(session, &collectionItem, "Imported"); err != nil {
		return derp.Wrap(err, location, "Saving imported CollectionItem")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported CollectionItem from the database
func (service *CollectionItem) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.CollectionItem.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Deleting record", importItem.LocalID)
	}

	return nil
}
