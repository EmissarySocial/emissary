package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KeyPackage struct {
	KeyPackageID primitive.ObjectID `bson:"_id"`
	UserID       primitive.ObjectID `bson:"userId"`
	MediaType    string             `bson:"mediaType"`
	Encoding     string             `bson:"encoding"`
	Content      string             `bson:"content"`
	Generator    string             `bson:"generator"`

	journal.Journal `bson:",inline"`
}

func NewKeyPackage() KeyPackage {
	return KeyPackage{
		KeyPackageID: primitive.NewObjectID(),
	}
}

/******************************
 * data.Object Interface
 ******************************/

func (keyPackage *KeyPackage) ID() string {
	return keyPackage.KeyPackageID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Stream.
// It is part of the AccessLister interface
func (keyPackage KeyPackage) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Stream
// It is part of the AccessLister interface
func (keyPackage KeyPackage) IsAuthor(userID primitive.ObjectID) bool {
	return userID == keyPackage.UserID
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (keyPackage KeyPackage) IsMyself(_ primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (keyPackage KeyPackage) RolesToGroupIDs(roles ...string) Permissions {
	return defaultRolesToGroupIDs(keyPackage.UserID, roles...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (keyPackage KeyPackage) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}
