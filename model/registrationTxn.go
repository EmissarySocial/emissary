package model

import (
	"net/mail"
	"time"

	"github.com/benpate/rosetta/convert"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

	// Internal values assigned by the server
	UserID string `form:"-"` // optional: Unique identifier for the User to be created.  Prevents replay/reuse attacks
}

// NewRegistrationTxn returns a fully initialized RegistrationTxn object
func NewRegistrationTxn() RegistrationTxn {
	return RegistrationTxn{
		UserID: primitive.NewObjectID().Hex(),
	}
}

func ParseRegistrationFromClaims(claims jwt.MapClaims) RegistrationTxn {
	return RegistrationTxn{
		UserID:         convert.String(claims["userId"]),
		DisplayName:    convert.String(claims["displayName"]),
		EmailAddress:   convert.String(claims["emailAddress"]),
		Username:       convert.String(claims["username"]),
		Password:       convert.String(claims["password"]),
		StateID:        convert.String(claims["stateId"]),
		InboxTemplate:  convert.String(claims["inboxTemplate"]),
		OutboxTemplate: convert.String(claims["outboxTemplate"]),
		AddGroups:      convert.String(claims["addGroups"]),
		RemoveGroups:   convert.String(claims["removeGroups"]),
		Secret:         convert.String(claims["secret"]),
	}
}

func (txn RegistrationTxn) Claims() jwt.MapClaims {

	// Create a new JWT token that expires in 24 hours
	now := time.Now()
	exp := now.Add(time.Hour * 24)

	// Create a jwt.Claims wtih required fields
	result := jwt.MapClaims{
		"userId":       txn.UserID,
		"displayName":  txn.DisplayName,
		"emailAddress": txn.EmailAddress,
		"username":     txn.Username,
		"iat":          now.Unix(),
		"exp":          exp.Unix(),
	}

	if txn.Password != "" {
		result["password"] = txn.Password
	}

	if txn.StateID != "" {
		result["stateId"] = txn.StateID
	}

	if txn.InboxTemplate != "" {
		result["inboxTemplate"] = txn.InboxTemplate
	}

	if txn.OutboxTemplate != "" {
		result["outboxTemplate"] = txn.OutboxTemplate
	}

	if txn.AddGroups != "" {
		result["addGroups"] = txn.AddGroups
	}

	if txn.RemoveGroups != "" {
		result["removeGroups"] = txn.RemoveGroups
	}

	if txn.Secret != "" {
		result["secret"] = txn.Secret
	}

	return result
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

	// Username is required
	if txn.Username == "" {
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

	// UserID must be a primitive.ObjectID
	if _, err := primitive.ObjectIDFromHex(txn.UserID); err != nil {
		return false
	}

	// Otherwise, rock on.
	return true
}

// IsInvalid is the inverse of `IsValid`.  It returns TRUE if the transaction is NOT VALID
func (txn RegistrationTxn) IsInvalid(secret string) bool {
	return !txn.IsValid(secret)
}
