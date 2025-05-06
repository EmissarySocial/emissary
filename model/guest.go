package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Guest represents an individual what has guestd content or authenticated via their email/username
type Guest struct {
	GuestID         primitive.ObjectID `bson:"_id"`             // Unique ID for the Guest
	Name            string             `bson:"fullName"`        // Full name of the Guest
	FediverseHandle string             `bson:"fediverseHandle"` // Fediverse username of the Guest (@user@server.social) used to send product notifications
	EmailAddress    string             `bson:"emailAddress"`    // Email address of the Guest
	RemoteIDs       mapof.String       `bson:"remoteIds"`       // Remote IDs of the Guest (e.g. PayPal, Stripe, etc.)

	// Embed journal to track changes
	journal.Journal `bson:",inline"`
}

func NewGuest() Guest {
	return Guest{
		GuestID: primitive.NewObjectID(),
	}
}

func (guest Guest) ID() string {
	return guest.GuestID.Hex()
}

func (guest Guest) Fields() []string {
	return []string{
		"_id",
		"name",
		"emailAddress",
		"fediverseId",
	}
}

func (guest *Guest) Update(emailAddress string, merchantAccountType string, remoteID string) bool {

	updated := false

	if guest.EmailAddress != emailAddress {
		guest.Name = emailAddress
		updated = true
	}

	if currentID := guest.RemoteIDs[merchantAccountType]; currentID != remoteID {
		guest.RemoteIDs[merchantAccountType] = remoteID
		updated = true
	}

	return updated
}
