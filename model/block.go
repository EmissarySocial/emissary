package model

import (
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Block represents many kinds of filters that are applied to messages before they are added into a User's inbox
type Block struct {
	BlockID     primitive.ObjectID `json:"blockId"     bson:"_id"`         // Unique identifier of this Block
	UserID      primitive.ObjectID `json:"userId"      bson:"userId"`      // Unique identifier of the User who owns this Block
	Type        string             `json:"type"        bson:"type"`        // Type of Block (e.g. "ACTOR", "ACTIVITY", "OBJECT")
	Label       string             `json:"label"       bson:"label"`       // Human-friendly label for this block
	Trigger     string             `json:"trigger"     bson:"trigger"`     // Parameter for this block type)
	Comment     string             `json:"comment"     bson:"comment"`     // Optional comment describing why this block exists
	IsActive    bool               `json:"isActive"    bson:"isActive"`    // If TRUE, this block is active and should be applied to incoming messages
	IsPublic    bool               `json:"isPublic"    bson:"isPublic"`    // If TRUE, this record is visible publicly
	Origin      OriginLink         `json:"origin"      bson:"origin"`      // Internal or External service where this block originated (used for subscriptions)
	PublishDate int64              `json:"publishDate" bson:"publishDate"` // Unix timestamp when this block was published to followers
	JSONLD      mapof.Any          `json:"jsonld"      bson:"jsonld"`      // JSON-LD data for this object

	journal.Journal `json:"-" bson:",inline"`
}

func NewBlock() Block {
	return Block{
		BlockID:  primitive.NewObjectID(),
		IsActive: true,
		IsPublic: true,
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
		"type",
		"trigger",
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

/******************************************
 * ActivityStreams Methods
 ******************************************/

// GetJSONLD returns a map document that conforms to the ActivityStreams 2.0 spec.
// This map will still need to be marshalled into JSON
func (block Block) GetJSONLD() mapof.Any {
	return block.JSONLD
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (block Block) PublishDateRCF3339() string {
	return time.Unix(block.PublishDate, 0).Format(time.RFC3339)
}
