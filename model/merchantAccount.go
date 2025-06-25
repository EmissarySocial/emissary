package model

import (
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MerchantAccount represents a User's account with a specific payment service.
// This account will be accessed on the User's behalf to charge Purchases for their merchant accounts
type MerchantAccount struct {
	MerchantAccountID    primitive.ObjectID `bson:"_id"`                  // Unique ID for the payment processor connection
	ConnectionID         primitive.ObjectID `bson:"connectionId"`         // Unique ID of the Connection that this MerchantAccount uses to access the payment processor
	UserID               primitive.ObjectID `bson:"userId"`               // Unique ID of the user who owns the account with this payment processor
	Type                 string             `bson:"type"`                 // Internal identifier of the payment processor (STRIPE, PAYPAL, etc.)
	Name                 string             `bson:"name"`                 // Human-friendly name for the payment processor account
	Description          string             `bson:"description"`          // Human-friendly Description of the payment processor account
	Vault                Vault              `bson:"vault" json:"-"`       // Vault data that is stored in the database (encrypted)
	Plaintext            mapof.String       `bson:"plaintext"`            // Plaintext data that is stored in the database (not encrypted)
	APIKeyExpirationDate int64              `bson:"apiKeyExpirationDate"` // Expiration date of the API key
	LiveMode             bool               `bson:"liveMode"`             // True if this is a live account, false if it is a test/sandbox account

	// Embed journal to track changes
	journal.Journal `bson:",inline"`
}

// NewMerchantAccount returns a fully initialized MerchantAccount object
func NewMerchantAccount() MerchantAccount {
	return MerchantAccount{
		MerchantAccountID: primitive.NewObjectID(),
		Vault:             NewVault(),
		Plaintext:         mapof.NewString(),
	}
}

// ID returns the unique ID of this MerchantAccount
func (merchantAccount MerchantAccount) ID() string {
	return merchantAccount.MerchantAccountID.Hex()
}

func (merchantAccount MerchantAccount) Fields() []string {
	return []string{
		"_id",
		"type",
		"name",
		"description",
		"liveMode",
	}
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this MerchantAccount.
// It is part of the AccessLister interface
func (merchantAccount *MerchantAccount) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this MerchantAccount
// It is part of the AccessLister interface
func (merchantAccount *MerchantAccount) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (merchantAccount *MerchantAccount) IsMyself(userID primitive.ObjectID) bool {
	return merchantAccount.UserID == userID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (merchantAccount *MerchantAccount) RolesToGroupIDs(roleIDs ...string) id.Slice {
	return nil
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (merchantAccount *MerchantAccount) RolesToPrivilegeIDs(roleIDs ...string) id.Slice {
	return nil
}

/******************************************
 * API URL Getters
 ******************************************/

// ProductURL returns the URL to the product page for this MerchantAccount.
func (merchantAccount MerchantAccount) ProductURL() string {

	switch merchantAccount.Type {

	// case ConnectionProviderPayPal:
	//	return "https://www.paypal.com/business/products"

	case ConnectionProviderStripe:
		return "https://dashboard.stripe.com/products?active=true"

	case ConnectionProviderStripeConnect:
		return "https://dashboard.stripe.com/products?active=true"
	}

	return ""
}

// APIKeyURL returns the URL to the API key management page for this MerchantAccount.
func (merchantAccount MerchantAccount) APIKeyURL() string {

	switch merchantAccount.Type {

	// case ConnectionProviderPayPal:
	//	return "https://www.paypal.com/business/keys"

	case ConnectionProviderStripe:
		return "https://dashboard.stripe.com/apikeys"

	case ConnectionProviderStripeConnect:
		return "https://dashboard.stripe.com/apikeys"
	}

	return ""
}

// HelpURL returns the URL to the help page for this MerchantAccount.
func (merchantAccount MerchantAccount) HelpURL() string {

	switch merchantAccount.Type {

	// case ConnectionProviderPayPal:
	//	return "https://emissary.dev/paypal"

	case ConnectionProviderStripe:
		return "https://emissary.dev/stripe"

	case ConnectionProviderStripeConnect:
		return "https://emissary.dev/stripe"
	}

	return ""
}

/******************************************
 * Other Methods
 ******************************************/

func (merchantAccount MerchantAccount) LookupCode() form.LookupCode {

	return form.LookupCode{
		Value:       merchantAccount.MerchantAccountID.Hex(),
		Label:       merchantAccount.Name,
		Description: merchantAccount.Description,
	}
}
