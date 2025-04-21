package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MerchantAccount represents a User's account with a specific payment service.
// This account will be accessed on the User's behalf to charge Subscribers for their merchant accounts
type MerchantAccount struct {
	MerchantAccountID    primitive.ObjectID `bson:"_id"`                  // Unique ID for the payment processor connection
	Type                 string             `bson:"type"`                 // Internal identifier of the payment processor (STRIPE, PAYPAL, etc.)
	UserID               primitive.ObjectID `bson:"userId"`               // Unique ID of the user who owns the account with this payment processor
	Name                 string             `bson:"name"`                 // Human-friendly name for the payment processor account
	Description          string             `bson:"description"`          // Human-friendly Description of the payment processor account
	Vault                Vault              `bson:"vault" json:"-"`       // Vault data that is stored in the database (encrypted)
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
	}
}

/******************************************
 * RoleStateEnumerator Interface
 ******************************************/

// State returns the current state of this object.
// For merchant accounts, there is no state, so it returns ""
func (merchantAccount MerchantAccount) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization.
// Since Rule records should only be accessible by the rule owner, this
// function only returns MagicRoleMyself if applicable.  Others (like Anonymous
// and Authenticated) should never be allowed on an Rule record, so they
// are not returned.
func (merchantAccount MerchantAccount) Roles(authorization *Authorization) []string {

	// Rules are private, so only 'myself' and 'owner' are allowed
	if authorization.UserID == merchantAccount.UserID {
		return []string{MagicRoleMyself, MagicRoleOwner}
	}

	// Intentionally NOT allowing MagicRoleAnonymous, or MagicRoleAuthenticated
	return []string{MagicRoleOwner}
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
