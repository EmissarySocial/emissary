package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

type Import struct {
	ImportID         primitive.ObjectID `bson:"_id"`              // Unique identifier for this Import record
	UserID           primitive.ObjectID `bson:"userId"`           // User profile that we're importing INTO
	StateID          string             `bson:"stateId"`          // Current state of this import process
	SourceID         string             `bson:"sourceId"`         // URL or Handle of the account being migrated
	SourceOAuthURL   string             `bson:"sourceOAuthURL"`   // OAuth 2.0 Authorization Endpoint to use when authorizing this import.
	SourceOAuthState string             `bson:"sourceOAuthState"` // State value passed to the OAuth server
	StateDescription string             `bson:"stateDescription"` // Human-friendly description of an error that has stopped this import.
	OAuthToken       oauth2.Token

	journal.Journal `bson:",inline"`
}

func NewImport() Import {
	return Import{
		ImportID:         primitive.NewObjectID(),
		StateID:          ImportStateNew,
		SourceOAuthState: primitive.NewObjectID().Hex(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (record Import) ID() string {
	return record.ImportID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Following.
// It is part of the AccessLister interface
func (record *Import) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Following
// It is part of the AccessLister interface
func (record *Import) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (record *Import) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == record.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (record *Import) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(record.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (record *Import) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}
