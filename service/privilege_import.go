package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported Privilege to the new profile.
func (service *Privilege) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.Privilege.Import"

	// Unmarshal the JSON document into a new Privilege
	privilege := model.NewPrivilege()
	if err := json.Unmarshal(document, &privilege); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Update mapping values in the importItem
	importItem.RemoteID = privilege.PrivilegeID
	importItem.LocalID = primitive.NewObjectID()

	// Map values from the original Privilege into the new, local Privilege
	privilege.PrivilegeID = importItem.LocalID   // Use the new localID for this record
	privilege.IdentityID = primitive.NilObjectID // This will be recalculated on save

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &privilege.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID", "UserID: "+user.UserID.Hex()+", PrivilegeID: "+privilege.PrivilegeID.Hex()))
	}

	// Map the CircleID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &privilege.CircleID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map CircleID", "UserID: "+user.UserID.Hex()+", CircleID: "+privilege.CircleID.Hex()))
	}

	// Map the MerchantAccountID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &privilege.MerchantAccountID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map MerchantAccountID", "UserID: "+user.UserID.Hex()+", MerchantAccountID: "+privilege.MerchantAccountID.Hex()))
	}

	// Map the ProductID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &privilege.ProductID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map ProductID", "UserID: "+user.UserID.Hex()+", ProductID: "+privilege.ProductID.Hex()))
	}

	// Save the Privilege to the database
	if err := service.Save(session, &privilege, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported Privilege")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported Privilege from the database
func (service *Privilege) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.Privilege.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
