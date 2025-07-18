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
	IdentityID        primitive.ObjectID `bson:"_id"`                  // Unique ID for the Identity
	Name              string             `bson:"name"`                 // Full name of the Individual ("John Connor")
	IconURL           string             `bson:"iconUrl"`              // URL to an icon representing the Identity (e.g., a profile picture)
	EmailAddress      string             `bson:"emailAddress"`         // Email address of the Identity ("john@connor.mil")
	WebfingerUsername string             `bson:"webfingerUsername"`    // Webfinger Username (e.g., "@john@connor.mil")
	ActivityPubActor  string             `bson:"activityPubActor"`     // ActivityPub Actor URL (https://connor.mil/@john) possibly derived from a WebFinger handle
	PrivilegeIDs      id.Slice           `bson:"privileges,omitempty"` // List of privileges associated with this Identity, either a circleID, or a remoteProductID

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
		"activityPubUsername",
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
	return !identityID.IsZero() && identity.IdentityID == identityID
}

// RolesToGroupIDs returns a map of RoleIDs to GroupIDs
func (identity Identity) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(identity.IdentityID, roleIDs...)
}

// RolesToPrivilegeIDs returns a map of RoleIDs to Privilege strings
func (identity Identity) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Getters
 ******************************************/

func (identity Identity) IsEmpty() bool {
	return (identity.EmailAddress == "") && (identity.WebfingerUsername == "")
}

// HasEmailAddress returns TRUE if the Identity has an email address.
func (identity Identity) HasEmailAddress() bool {
	return identity.EmailAddress != ""
}

// HasActivityPubActor TRUE if the Identity has an ActivityPub Actor.
func (identity Identity) HasActivityPubActor() bool {
	return identity.ActivityPubActor != ""
}

// HasWebfingerUsername TRUE if the Identity has a Webfinger Username.
func (identity Identity) HasWebfingerUsername() bool {
	return identity.WebfingerUsername != ""
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

func (identity *Identity) Identifier(identifierType string) string {

	switch identifierType {

	case IdentifierTypeEmail:
		return identity.EmailAddress

	case IdentifierTypeActivityPub:
		return identity.ActivityPubActor

	case IdentifierTypeWebfinger:
		return identity.WebfingerUsername
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

	case IdentifierTypeWebfinger:
		identity.WebfingerUsername = value
		identity.ActivityPubActor = "" // This extra step resets the ActivityPub Actor so that we can recalculate it on save.
		return true
	}

	return false
}

func (identity *Identity) RemoveIdentifier(identifierType string, value string) bool {

	switch identifierType {

	case IdentifierTypeEmail:
		if identity.EmailAddress == value {
			identity.EmailAddress = ""
			return true
		}

	case IdentifierTypeActivityPub:
		if identity.ActivityPubActor == value {
			identity.ActivityPubActor = ""
			identity.WebfingerUsername = ""
			return true
		}

	case IdentifierTypeWebfinger:
		if identity.WebfingerUsername == value {
			identity.ActivityPubActor = ""
			identity.WebfingerUsername = ""
			return true
		}
	}

	return false
}

// HasPrivilege returns TRUE if the Identity has any of the provided privileges.
func (identity Identity) HasPrivilege(privilege ...primitive.ObjectID) bool {
	return identity.PrivilegeIDs.ContainsAny(privilege...)
}

// SetPrivilegeID adds a privilegeID to the Identity's list of privileges.
func (identity *Identity) SetPrivilegeID(privilegeID primitive.ObjectID) {

	// RULE: Identity must not be nil
	if identity == nil {
		return
	}

	// RULE: Do not set Zero privilegeIDs
	if privilegeID.IsZero() {
		return
	}

	// RULE: If the Identity already has this privilege, do nothing.
	if identity.HasPrivilege(privilegeID) {
		return
	}

	// Add the privilegeID to the Identity
	identity.PrivilegeIDs = append(identity.PrivilegeIDs, privilegeID)
}

// RemovePrivilegeID safely removes a privilegeID from the Identity's list of privileges.
func (identity *Identity) RemovePrivilegeID(privilegeID primitive.ObjectID) {
	// RULE: Identity must not be nil
	if identity == nil {
		return
	}

	// RULE: Do not remove Zero privilegeIDs
	if privilegeID.IsZero() {
		return
	}

	// Remove the privilegeID from the Identity
	index := identity.PrivilegeIDs.IndexOf(privilegeID)

	if index >= 0 {
		identity.PrivilegeIDs = append(identity.PrivilegeIDs[:index], identity.PrivilegeIDs[index+1:]...)
	}
}
