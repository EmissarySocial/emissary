package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Host is a Domain Object that defines a particular Host domain.
type Host struct {
	HostID  primitive.ObjectID `json:"HostId" bson:"_id"`        // Unique ID of this Host
	OwnerID primitive.ObjectID `json:"ownerId"   bson:"ownerId"` // Unique ID of the single owner of this Host.
	Domains []string           `json:"domains"   bson:"domains"` // Array of domain names that can be used for this domain name.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key for this record
func (host *Host) ID() string {
	return host.HostID.Hex()
}
