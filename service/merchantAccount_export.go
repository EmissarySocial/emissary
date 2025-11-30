package service

import (
	"encoding/json"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (service *MerchantAccount) ExportCollection(session data.Session, userID primitive.ObjectID) ([]model.IDOnly, error) {
	criteria := exp.Equal("userId", userID)
	return service.QueryIDOnly(session, criteria, option.SortAsc("createDate"))
}

func (service *MerchantAccount) ExportDocument(session data.Session, userID primitive.ObjectID, merchantAccountID primitive.ObjectID) (string, error) {

	const location = "service.User.ExportDocument"

	// Load the User
	merchantAccount := model.NewMerchantAccount()
	if err := service.LoadByUserAndID(session, userID, merchantAccountID, &merchantAccount); err != nil {
		return "", derp.Wrap(err, location, "Unable to load MerchantAccount", merchantAccount.MerchantAccountID)
	}

	// Get Connection Type
	connection := model.NewConnection()
	if err := service.connectionService.LoadByID(session, merchantAccount.ConnectionID, &connection); err != nil {
		return "", derp.Wrap(err, location, "Unable to load connection", merchantAccount.ConnectionID)
	}

	// Decrypt Vault values
	decryptedVault, err := service.DecryptVault(&merchantAccount)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to decrypt vault")
	}

	export := mapof.Any{
		"MerchantAcccountID":   merchantAccount.MerchantAccountID.Hex(),
		"ConnectionID":         connection.Type,
		"UserID":               merchantAccount.UserID,
		"Type":                 merchantAccount.Type,
		"Name":                 merchantAccount.Name,
		"Description":          merchantAccount.Description,
		"Vault":                decryptedVault,
		"Plaintext":            merchantAccount.Plaintext,
		"APIKeyExpirationDate": merchantAccount.APIKeyExpirationDate,
		"LiveMode":             merchantAccount.LiveMode,
		"Journal":              merchantAccount.Journal,
	}

	// Marshal the user as JSON
	result, err := json.Marshal(export)

	if err != nil {
		return "", derp.Wrap(err, location, "Unable to marshal MerchantAccount", merchantAccount)
	}

	// Success
	return string(result), nil
}
