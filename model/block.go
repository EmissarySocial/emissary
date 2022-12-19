package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BlockSourceInternal represents a block that was created directly by the owner
const BlockSourceInternal = "INTERNAL"

// BlockSourceActivityPub represents a block that was created by an external ActivityPub server
const BlockSourceActivityPub = "ACTIVITYPUB"

// BlockTypeURL blocks all messages that link to a specific domain or URL prefix
const BlockTypeURL = "URL"

// BlockTypeUser blocks all messages from a specific user
const BlockTypeActor = "ACTOR"

// BlockTypeUser blocks all messages that contain a particular phrase (hashtag)
const BlockTypeContent = "CONTENT"

// BlockTypeExternal passes messages to an external block service (TBD) for analysis.
const BlockTypeExternal = "EXTERNAL"

// Block represents many kinds of filters that are applied to messages before they are added into a User's inbox
type Block struct {
	BlockID  primitive.ObjectID `path:"blockId" json:"blockId" bson:"_id"`        // Unique identifier of this Block
	UserID   primitive.ObjectID `path:"userId"  json:"userId"  bson:"userId"`     // Unique identifier of the User who owns this Block
	Source   string             `path:"source"  json:"source"  bson:"source"`     // Source of the Block (e.g. "INTERNAL", "ACTIVITYPUB")
	Type     string             `path:"type"    json:"type"    bson:"type"`       // Type of Block (e.g. "ACTOR", "ACTIVITY", "OBJECT")
	Trigger  string             `path:"trigger" json:"trigger" bson:"trigger"`    // Parameter for this block type)
	Comment  string             `path:"comment" json:"comment" bson:"comment"`    // Optional comment describing why this block exists
	IsPublic bool               `path:"isPublic" json:"isPublic" bson:"isPublic"` // If TRUE, this record is visible publicly
	IsActive bool               `path:"isActive" json:"isActive" bson:"isActive"` // If TRUE, this record is active

	journal.Journal `path:"journal" json:"-" bson:"journal"`
}

func NewBlock() Block {
	return Block{
		BlockID: primitive.NewObjectID(),
	}
}

func BlockSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"blockId":  schema.String{Format: "objectId"},
			"userId":   schema.String{Format: "objectId"},
			"source":   schema.String{Enum: []string{BlockSourceInternal, BlockSourceActivityPub}},
			"type":     schema.String{Enum: []string{BlockTypeURL, BlockTypeActor, BlockTypeContent, BlockTypeExternal}},
			"trigger":  schema.String{},
			"comment":  schema.String{},
			"isPublic": schema.Boolean{},
			"isActive": schema.Boolean{},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (block Block) ID() string {
	return block.BlockID.Hex()
}
