package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Product to the new profile.
func (service *Product) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Product.Import"

	// Unmarshal the JSON document into a new Product
	product := model.NewProduct()
	if err := json.Unmarshal(document, &product); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = product.ProductID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Product into the new, local Product
	product.ProductID = importItem.LocalID // Use the new localID for this record

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &product.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", ProductID: "+product.ProductID.Hex()))
	}

	// Map the MerchantAccountID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &product.MerchantAccountID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map MerchantAccountID", "UserID: "+user.UserID.Hex()+", ProductID: "+product.ProductID.Hex()))
	}

	// Save the Product to the database
	if err := service.Save(session, &product, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Product")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Product from the database
func (service *Product) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Product.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
