package model

import (
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Identity represents a combination of identifiers that all represent a single individual.
// This is used to track pseud-logins by individuals who do not have a registered username on this server.
// Identities can be tied to a Follower and to a Purchase via the `Identifier` type.
type Identity struct {
	IdentityID              primitive.ObjectID `bson:"_id"`                               // Unique ID for the Identity
	Name                    string             `bson:"name"`                              // Full name of the Individual ("John Connor")
	IconURL                 string             `bson:"iconUrl"`                           // URL to an icon representing the Identity (e.g., a profile picture)
	EmailAddress            string             `bson:"emailAddress"`                      // Email address of the Identity ("john@connor.mil")
	WebFingerHandle         string             `bson:"webfingerHandle"`                   // WebFinger handle of the Identity ("@john@connor.social")
	EmailVerifiedDate       int64              `bson:"emailVerifiedDate"`                 // Unix epoch (in seconds) when the email address was verified
	WebFingerVerifiedDate   int64              `bson:"webfingerVerifiedDate"`             // Unix epoch (in seconds) when the WebFinger handle was verified
	VerificationSecret      string             `bson:"verificationSecret,omitempty"`      // Secret token used to verify email or fediverse values
	VerificationExpiresDate int64              `bson:"verificationExpiresDate,omitempty"` // Unix epoch (in seconds) when the verification secret expires and must be regenerated
	Privileges              sliceof.String     `bson:"privileges,omitempty"`              // List of privileges associated with this Identity, either a circleID, or a remoteProductID

	// Embed journal to track changes
	journal.Journal `bson:",inline"`
}

// NewIdentity returns a fully populated Identity object
func NewIdentity() Identity {
	return Identity{
		IdentityID: primitive.NewObjectID(),
	}
}

// ID returns the unique identifier for this Identity as a string.
func (identity Identity) ID() string {
	return identity.IdentityID.Hex()
}

// Fields returns a list of field names that are used to identify this Identity.
func (identity Identity) Fields() []string {
	return []string{
		"_id",
		"name",
		"iconUrl",
		"emailAddress",
		"webFingerHandle",
	}
}

// SetIdentifier sets the value of the provided identifier type.
// This method returns TRUE if the identifier type is recognized.
func (identity *Identity) SetIdentifier(identifierType string, value string) bool {

	switch identifierType {

	case IdentifierTypeEmail:
		identity.EmailAddress = value
		return true

	case IdentifierTypeWebFinger:
		identity.WebFingerHandle = value
		return true
	}

	return false
}

// IsVerified returns TRUE if the provided identifier type has been verified.
func (identity *Identity) IsVerified(identifierType string) bool {

	switch identifierType {

	case IdentifierTypeEmail:
		return identity.EmailVerifiedDate > 0

	case IdentifierTypeWebFinger:
		return identity.WebFingerVerifiedDate > 0
	}

	return false
}

// Verify marks the selected identifier as "verified" by setting the verification date to the current time.
// If the identifier has already been verified, then this method does nothing and returns false.
func (identity *Identity) Verify(identifierType string) bool {

	switch identifierType {

	case IdentifierTypeEmail:
		if identity.EmailVerifiedDate == 0 {
			identity.EmailVerifiedDate = time.Now().Unix()
			return true
		}

	case IdentifierTypeWebFinger:
		if identity.WebFingerVerifiedDate == 0 {
			identity.WebFingerVerifiedDate = time.Now().Unix()
			return true
		}
	}

	return false
}

// HasPrivilege returns TRUE if the Identity has any of the provided privileges.
func (identity Identity) HasPrivilege(privilege ...string) bool {
	return identity.Privileges.ContainsAny(privilege...)
}
