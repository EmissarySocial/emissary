package model

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Identity represents a combination of identifiers that all represent a single individual.
// This is used to track pseud-logins by individuals who do not have a registered username on this server.
// Identities can be tied to a Follower and to a Privilege via the two identifiers: EmailAddress and WebfingerHandle.
type Identity struct {
	IdentityID            primitive.ObjectID `bson:"_id"`                   // Unique ID for the Identity
	Name                  string             `bson:"name"`                  // Full name of the Individual ("John Connor")
	IconURL               string             `bson:"iconUrl"`               // URL to an icon representing the Identity (e.g., a profile picture)
	EmailAddress          string             `bson:"emailAddress"`          // Email address of the Identity ("john@connor.mil")
	WebfingerHandle       string             `bson:"webfingerHandle"`       // WebFinger handle of the Identity ("@john@connor.social")
	EmailVerifiedDate     int64              `bson:"emailVerifiedDate"`     // Unix epoch (in seconds) when the email address was verified
	WebfingerVerifiedDate int64              `bson:"webfingerVerifiedDate"` // Unix epoch (in seconds) when the WebFinger handle was verified
	Privileges            sliceof.String     `bson:"privileges,omitempty"`  // List of privileges associated with this Identity, either a circleID, or a remoteProductID

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
		"webfingerHandle",
	}
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of the object.
func (identity Identity) State() string {
	return ""
}

// IsAuthor returns TRUE if the provided UserID the author of this object
func (identity Identity) IsAuthor(primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
func (identity Identity) IsMyself(identityID primitive.ObjectID) bool {
	return identity.IdentityID == identityID
}

// RolesToGroupIDs returns a map of RoleIDs to GroupIDs
func (identity Identity) RolesToGroupIDs(...string) id.Slice {
	return id.NewSlice()
}

// RolesToPrivileges returns a map of RoleIDs to Privilege strings
func (identity Identity) RolesToPrivileges(...string) sliceof.String {
	return sliceof.NewString()
}

/******************************************
 * Other Getters
 ******************************************/

func (identity Identity) HasEmailAddress() bool {
	return identity.EmailAddress != ""
}

func (identity Identity) HasWebfingerHandle() bool {
	return identity.WebfingerHandle != ""
}

// Icon returns an icon name to use for this Identity, based on the type of identifier(s) present.
func (identity Identity) Icon() string {

	if identity.WebfingerHandle != "" {
		return "fediverse"
	}

	if identity.EmailAddress != "" {
		return "email"
	}

	return "person-circle"
}

// IdentifierType returns the "primary" identifier type that is present in this Identity.
func (identity Identity) IdentifierType() string {

	if identity.WebfingerHandle != "" {
		return IdentifierTypeWebFinger
	}

	if identity.EmailAddress != "" {
		return IdentifierTypeEmail
	}

	return ""
}

// Identifier returns the "primary" identifier for this Identity.
func (identity Identity) Identifier() string {

	if identity.WebfingerHandle != "" {
		return identity.WebfingerHandle
	}

	if identity.EmailAddress != "" {
		return identity.EmailAddress
	}

	return ""
}

// SetIdentifier sets the value of the provided identifier type.
// This method returns TRUE if the identifier type is recognized.
func (identity *Identity) SetIdentifier(identifierType string, value string) bool {

	switch identifierType {

	case IdentifierTypeEmail:
		identity.EmailAddress = value
		return true

	case IdentifierTypeWebFinger:
		identity.WebfingerHandle = value
		return true
	}

	return false
}

// IsVerified returns TRUE if the provided identifier type has been verified.
func (identity Identity) IsVerified(identifierType string) bool {

	switch identifierType {

	case IdentifierTypeEmail:
		return identity.EmailVerifiedDate > 0

	case IdentifierTypeWebFinger:
		return identity.WebfingerVerifiedDate > 0
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
		if identity.WebfingerVerifiedDate == 0 {
			identity.WebfingerVerifiedDate = time.Now().Unix()
			return true
		}
	}

	return false
}

// HasPrivilege returns TRUE if the Identity has any of the provided privileges.
func (identity Identity) HasPrivilege(privilege ...string) bool {
	return identity.Privileges.ContainsAny(privilege...)
}
