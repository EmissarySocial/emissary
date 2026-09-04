package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/delta"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * User Connection
 *
 * One User's connection to an external service that acts on their behalf.
 * Secrets live in the Vault, which never exports; everything else -- remote
 * IDs, routing, provider settings -- lives in Data. `IsActive` is the User's
 * own switch, and its CHANGE is what installs or removes the connection at
 * the remote service. See MAILING-LISTS.md.
 ******************************************/

// UserConnection represents one User's connection to a single external service
type UserConnection struct {
	UserConnectionID primitive.ObjectID `bson:"_id"`      // Unique ID for this connection
	UserID           primitive.ObjectID `bson:"userId"`   // Unique ID of the User who owns this connection
	Type             string             `bson:"type"`     // Internal identifier of the remote service (MAILCHIMP, etc.)
	IsActive         delta.Bool         `bson:"isActive"` // TRUE if this connection should be installed at the remote service
	Status           string             `bson:"status"`   // Health of the connection, one of the UserConnectionStatus* constants
	Data             mapof.String       `bson:"data"`     // Non-secret settings and remote IDs
	Vault            Vault              `bson:"vault"`    // Secrets for this connection (encrypted at rest)

	// Embed journal to track changes
	journal.Journal `json:"-" bson:",inline"`
}

// NewUserConnection returns a fully initialized UserConnection object
func NewUserConnection() UserConnection {
	return UserConnection{
		UserConnectionID: primitive.NewObjectID(),
		IsActive:         delta.NewBool(false),
		Data:             mapof.NewString(),
		Vault:            NewVault(),
	}
}

// RULE: this type deliberately has NO Fields() projection. A projection that omitted
// `vault` or `data` would make a configured connection read as unconfigured -- silently
// stopping a sync, or comparing a presented secret against an empty stored one.

// ID returns the unique ID of this UserConnection
func (userConnection UserConnection) ID() string {
	return userConnection.UserConnectionID.Hex()
}

// IsReady returns TRUE if this connection is switched on and its credentials still work
func (userConnection UserConnection) IsReady() bool {

	if userConnection.IsActive.IsFalse() {
		return false
	}

	return userConnection.Status != UserConnectionStatusReconnect
}

// NeedsReconnect returns TRUE if the remote service has rejected this connection's credentials
func (userConnection UserConnection) NeedsReconnect() bool {
	return userConnection.Status == UserConnectionStatusReconnect
}

// Label returns a human-friendly name for the service this connection reaches
func (userConnection UserConnection) Label() string {

	switch userConnection.Type {

	case UserConnectionTypeMailchimp:
		return "Mailchimp"
	}

	return userConnection.Type
}

// Icon returns the name of the icon that represents this connection's service
func (userConnection UserConnection) Icon() string {

	switch userConnection.Type {

	case UserConnectionTypeMailchimp:
		return "email"
	}

	return "cloud"
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this UserConnection.
// It is part of the AccessLister interface
func (userConnection *UserConnection) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID is the author of this UserConnection.
// It is part of the AccessLister interface
func (userConnection *UserConnection) IsAuthor(_ primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID.
// It is part of the AccessLister interface
func (userConnection *UserConnection) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userConnection.UserID == userID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (userConnection *UserConnection) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(userConnection.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (userConnection *UserConnection) RolesToPrivilegeIDs(_ ...string) Permissions {
	return NewPermissions()
}
