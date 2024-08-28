package model

import "net/mail"

// RegistrationTsn represents the data that is submitted when a User registers for a new account
type RegistrationTxn struct {
	DisplayName    string `form:"displayName"`    // required: User's DisplayName
	EmailAddress   string `form:"emailAddress"`   // required: User's EmailAddress
	Username       string `form:"username"`       // optional: User's Username
	Password       string `form:"password"`       // optional: User's Password
	StateID        string `form:"stateId"`        // optional: User's StateID
	InboxTemplate  string `form:"inboxTemplate"`  // optional: User's InboxTemplate
	OutboxTemplate string `form:"outboxTemplate"` // optional: User's OutboxTemplate
	AddGroups      string `form:"addGroups"`      // optional: Comma-separated list of GroupIDs to add the User to
	RemoveGroups   string `form:"removeGroups"`   // optional: Comma-separated list of GroupIDs to remove the User from
	Secret         string `form:"secret"`         // optional: Secret key used to validate the registration
}

// NewRegistrationTxn returns a fully initialized RegistrationTxn object
func NewRegistrationTxn() RegistrationTxn {
	return RegistrationTxn{}
}

// IsValid returns TRUE if all of the required fields are present and valid.
// The "secret"  field is required, and if not empty, MUST match the `secret` value in the Transaction.
func (txn RegistrationTxn) IsValid(secret string) bool {

	// If the `secret` argument is present, then it must match the `secret`` in the Transaction
	if secret != "" {
		if txn.Secret != secret {
			return false
		}
	}

	// DisplayName is required
	if txn.DisplayName == "" {
		return false
	}

	// EmailAddress is required
	if txn.EmailAddress == "" {
		return false
	}

	// EmailAddress must parse as a valid email address
	if _, err := mail.ParseAddress(txn.EmailAddress); err != nil {
		return false
	}

	// Otherwise, rock on.
	return true
}

// IsInvalid is the inverse of `IsValid`.  It returns TRUE if the transaction is NOT VALID
func (txn RegistrationTxn) IsInvalid(secret string) bool {
	return !txn.IsValid(secret)
}
