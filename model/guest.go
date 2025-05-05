package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Guest represents an individual what has guestd content or authenticated via their email/username
type Guest struct {
	GuestID         primitive.ObjectID `bson:"_id"`             // Unique ID for the Guest
	Name            string             `bson:"fullName"`        // Full name of the Guest
	FediverseHandle string             `bson:"fediverseHandle"` // Fediverse username of the Guest (@user@server.social) used to send product notifications
	EmailAddress    string             `bson:"emailAddress"`    // Email address of the Guest

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

func (guest *Guest) UpdateWith(other *Guest) bool {

	updated := false

	if guest.Name != other.Name {
		guest.Name = other.Name
		updated = true
	}

	if guest.EmailAddress != other.EmailAddress {
		guest.EmailAddress = other.EmailAddress
		updated = true
	}

	if guest.FediverseHandle != other.FediverseHandle {
		guest.FediverseHandle = other.FediverseHandle
		updated = true
	}

	return updated
}
