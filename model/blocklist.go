package model

import (
	"time"

	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BlockList represents a group of blocked identities that can be published (via GitHub?) to other people
type BlockList struct {
	BlockListID primitive.ObjectID
	Label       string
	Description string

	Blocks map[string]BlockInfo
	Public bool
	Active bool
	journal.Journal
}

// BlockInfo represents the meta-data for any identity that has been blocked in this list.
type BlockInfo struct {
	Reason    string
	BlockDate int64
}

// ID returns the unique identifier for this blocklist
func (blocklist *BlockList) ID() string {
	return blocklist.BlockListID.Hex()
}

// Add adds an identity to this blocklist.  Returns TRUE if the item was added or updated.
func (blocklist *BlockList) Add(identity string, reason string) bool {

	// If we already have this identity in the Blocklist...
	if _, ok := blocklist.Blocks[identity]; ok {

		// ... and the reasons are identical
		if blocklist.Blocks[identity].Reason == reason {

			// Then there is nothing to change.  Return FALSE
			return false
		}
	}

	// There is SOMETHING to change.  So make the change, and return TRUE.
	blocklist.Blocks[identity] = BlockInfo{
		Reason:    reason,
		BlockDate: time.Now().Unix(),
	}

	return true
}

// Remove removes an identity from this blocklist.
func (blocklist *BlockList) Remove(identity string) {
	delete(blocklist.Blocks, identity)
}
