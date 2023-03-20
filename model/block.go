package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Block represents many kinds of filters that are applied to messages before they are added into a User's inbox
type Block struct {
	BlockID  primitive.ObjectID `json:"blockId" bson:"_id"`       // Unique identifier of this Block
	UserID   primitive.ObjectID `json:"userId"  bson:"userId"`    // Unique identifier of the User who owns this Block
	Type     string             `json:"type"    bson:"type"`      // Type of Block (e.g. "ACTOR", "ACTIVITY", "OBJECT")
	Trigger  string             `json:"trigger" bson:"trigger"`   // Parameter for this block type)
	Behavior string             `json:"behavior" bson:"behavior"` // Behavior for this block type (e.g. "BLOCK", "MUTE", "ALLOW")
	Comment  string             `json:"comment" bson:"comment"`   // Optional comment describing why this block exists
	IsPublic bool               `json:"isPublic" bson:"isPublic"` // If TRUE, this record is visible publicly
	Origin   OriginLink         `json:"origin" bson:"origin"`     // Internal or External service where this block originated (used for subscriptions)

	journal.Journal `json:"-" bson:"journal"`
}

func NewBlock() Block {
	return Block{
		BlockID: primitive.NewObjectID(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (block Block) ID() string {
	return block.BlockID.Hex()
}

func (block Block) Fields() []string {
	return []string{
		"_id",
		"userId",
		"type",
		"trigger",
		"comment",
		"isPublic",
		"isActive",
	}
}

/******************************************
 * RoleStateEnumerator Interface
 ******************************************/

// State returns the current state of this object.
// For users, there is no state, so it returns ""
func (block Block) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization.
// Since Block records should only be accessible by the block owner, this
// function only returns MagicRoleMyself if applicable.  Others (like Anonymous
// and Authenticated) should never be allowed on an Block record, so they
// are not returned.
func (block Block) Roles(authorization *Authorization) []string {

	// Folders are private, so only MagicRoleMyself is allowed
	if authorization.UserID == block.UserID {
		return []string{MagicRoleMyself}
	}

	// Intentionally NOT allowing MagicRoleAnonymous, MagicRoleAuthenticated, or MagicRoleOwner
	return []string{}
}
