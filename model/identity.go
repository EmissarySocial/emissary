package model

import (
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Identity represents a combination of identifiers that all represent a single individual.
// This is used to track pseud-logins by individuals who do not have a registered username on this server.
// Identities can be tied to a Follower and to a Privilege via the two identifiers: EmailAddress and ActivityPub.
type Identity struct {
	IdentityID       primitive.ObjectID `bson:"_id"`                  // Unique ID for the Identity
	Name             string             `bson:"name"`                 // Full name of the Individual ("John Connor")
	IconURL          string             `bson:"iconUrl"`              // URL to an icon representing the Identity (e.g., a profile picture)
	EmailAddress     string             `bson:"emailAddress"`         // Email address of the Identity ("john@connor.mil")
	ActivityPubActor string             `bson:"activityPubActor"`     // ActivityPub Actor URL (https://connor.mil/@john) possibly derived from a WebFinger handle
	PrivilegeIDs     id.Slice           `bson:"privileges,omitempty"` // List of privileges associated with this Identity, either a circleID, or a remoteProductID

	// Embed journal to track changes
	journal.Journal `bson:",inline"`
}

// NewIdentity returns a fully populated Identity object
func NewIdentity() Identity {
	return Identity{
		IdentityID:   primitive.NewObjectID(),
		PrivilegeIDs: id.NewSlice(),
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
		"activityPubActor",
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
	return nil
}

// RolesToPrivilegeIDs returns a map of RoleIDs to Privilege strings
func (identity Identity) RolesToPrivilegeIDs(...string) id.Slice {
	return nil
}

/******************************************
 * Other Getters
 ******************************************/

func (identity Identity) IsZero() bool {
	return identity.IdentityID.IsZero()
}

// HasEmailAddress returns TRUE if the Identity has an email address.
func (identity Identity) HasEmailAddress() bool {
	return identity.EmailAddress != ""
}

// HasActivityPubActor TRUE if the Identity has an ActivityPub Actor URL.
func (identity Identity) HasActivityPubActor() bool {
	return identity.ActivityPubActor != ""
}

// Icon returns an icon name to use for this Identity, based on the type of identifier(s) present.
func (identity Identity) Icon() string {

	if identity.HasActivityPubActor() {
		return "activitypub"
	}

	if identity.HasEmailAddress() {
		return "email"
	}

	return "person-circle"
}

// IdentifierType returns the "primary" identifier type that is present in this Identity.
func (identity Identity) IdentifierType() string {

	if identity.HasActivityPubActor() {
		return IdentifierTypeActivityPub
	}

	if identity.HasEmailAddress() {
		return IdentifierTypeEmail
	}

	return ""
}

// Identifier returns the "primary" identifier for this Identity.
func (identity Identity) Identifier() string {

	if identity.HasActivityPubActor() {
		return identity.ActivityPubActor
	}

	if identity.HasEmailAddress() {
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

	case IdentifierTypeActivityPub:
		identity.ActivityPubActor = value
		return true
	}

	return false
}

// HasPrivilege returns TRUE if the Identity has any of the provided privileges.
func (identity Identity) HasPrivilege(privilege ...primitive.ObjectID) bool {
	return identity.PrivilegeIDs.ContainsAny(privilege...)
}
