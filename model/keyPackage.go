package model

import (
	"github.com/EmissarySocial/emissary/tools/emojikey"
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// KeyPackage represents a published MLS KeyPackage that other actors can use to add this User to encrypted groups.
type KeyPackage struct {
	KeyPackageID  primitive.ObjectID `bson:"_id"`
	UserID        primitive.ObjectID `bson:"userId"`
	MediaType     string             `bson:"mediaType"`
	Encoding      string             `bson:"encoding"`
	Content       string             `bson:"content"`
	Summary       string             `bson:"summary"`
	GeneratorID   string             `bson:"generatorId"`
	Ciphersuite   string             `bson:"ciphersuite"`
	GeneratorName string             `bson:"generatorName"`

	journal.Journal `bson:",inline"`
}

// NewKeyPackage returns a fully initialized KeyPackage
func NewKeyPackage() KeyPackage {
	return KeyPackage{
		KeyPackageID: primitive.NewObjectID(),
	}
}

/******************************
 * data.Object Interface
 ******************************/

// ID returns the primary key of this KeyPackage as a hex-encoded string.
// It is part of the data.Object interface
func (keyPackage *KeyPackage) ID() string {
	return keyPackage.KeyPackageID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this KeyPackage.
// It is part of the AccessLister interface
func (keyPackage KeyPackage) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID is the author of this KeyPackage
// It is part of the AccessLister interface
func (keyPackage KeyPackage) IsAuthor(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == keyPackage.UserID
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
func (keyPackage KeyPackage) RolesToPrivilegeIDs(_ ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Methods
 ******************************************/

// EmojiKey returns the individual emojis (with display names) that make up this KeyPackage's Summary
func (keyPackage KeyPackage) EmojiKey() []emojikey.Emoji {
	return emojikey.Parse(keyPackage.Summary)
}
