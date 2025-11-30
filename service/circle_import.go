package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Circle to the new profile.
func (service *Circle) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Circle.Import"

	// Unmarshal the JSON document into a new Circle
	circle := model.NewCircle()
	if err := json.Unmarshal(document, &circle); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = circle.CircleID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Circle into the new, local Circle
	circle.CircleID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &circle.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", CircleID: "+circle.CircleID.Hex()))
	}

	// Map ProductIDs
	for index, productID := range circle.ProductIDs {
		if err := service.importItemService.mapRemoteID(session, user.UserID, &circle.ProductIDs[index]); err != nil {
			return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map ProductID", "UserID: "+user.UserID.Hex()+", CircleID: "+circle.CircleID.Hex(), "ProductID:"+productID.Hex()))
		}
	}

	// Save the Circle to the database
	if err := service.Save(session, &circle, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Circle")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Circle from the database
func (service *Circle) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Circle.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
