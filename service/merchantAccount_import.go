package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportSave is a part of the "Importable" interface, and saves an imported MerchantAccount to the new profile.
func (service *MerchantAccount) Import(session data.Session, _ *model.Import, importItem *model.ImportItem, user *model.User, document []byte) error {

	const location = "service.MerchantAccount.Import"

	// Unmarshal the JSON document into a new MerchantAccount
	remoteRecord := mapof.NewAny()
	if err := json.Unmarshal(document, &remoteRecord); err != nil {
		return derp.Wrap(err, location, "Unable to parse remote document", document)
	}

	// Parse the remote record ID
	remoteRecordID, err := primitive.ObjectIDFromHex(remoteRecord.GetString("MerchantAccountID"))

	if err != nil {
		return derp.Wrap(err, location, "Unable to parse remoteRecordID", remoteRecord.GetString("MerchantAccountID"))
	}

	// Update mapping values in the importItem
	importItem.RemoteID = remoteRecordID
	importItem.LocalID = primitive.NewObjectID()

	// Create a new MerchantAccount from the remote record
	merchantAccount := model.NewMerchantAccount()
	merchantAccount.MerchantAccountID = importItem.LocalID // Use the new localID for this record
	merchantAccount.Type = remoteRecord.GetString("Type")
	merchantAccount.Name = remoteRecord.GetString("Name")
	merchantAccount.Description = remoteRecord.GetString("Description")
	merchantAccount.Plaintext = remoteRecord.GetMapOfString("Plaintext")
	merchantAccount.APIKeyExpirationDate = remoteRecord.GetInt64("APIKeyExpirationDate")
	merchantAccount.LiveMode = remoteRecord.GetBool("LiveMode")

	// Map (unencrypted) items in Vault
	for key, value := range remoteRecord.GetMapOfString("Vault") {
		merchantAccount.Vault.SetString(key, value)
	}

	// Map the UserID
	if err := service.importItemService.mapRemoteID(session, user.UserID, &merchantAccount.UserID); err != nil {
		return derp.ReportAndReturn(derp.Wrap(err, location, "Unable to map UserID: "+user.UserID.Hex()))
	}

	// Save the MerchantAccount to the database
	if err := service.Save(session, &merchantAccount, "Imported"); err != nil {
		return derp.Wrap(err, location, "Unable to save imported MerchantAccount")
	}

	// A Man, A Plan, A Canal. Panama.
	return nil
}

// UndoImport is a part of the "Importable" interface, and deletes imported MerchantAccount from the database
func (service *MerchantAccount) UndoImport(session data.Session, importItem *model.ImportItem) error {

	const location = "service.MerchantAccount.UndoImport"

	if err := service.HardDeleteByID(session, importItem.UserID, importItem.LocalID); err != nil {
		return derp.Wrap(err, location, "Unable to delete record", importItem.LocalID)
	}

	return nil
}
