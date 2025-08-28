package model

import (
	dt "github.com/benpate/domain"
)

// ActorSummary is a record returned by the ActivityStream directory
type ActorSummary struct {
	ID       string `bson:"id"`
	Type     string `bson:"type"`
	Name     string `bson:"name"`
	Icon     string `bson:"icon"`
	Username string `bson:"preferredUsername"`
}

// UsernameOrID returns the best identifier we can find for an Actor:
// either the Actor' username, if it exists, or the Actor's ID
func (actor ActorSummary) UsernameOrID() string {
	if actor.Username != "" {
		return "@" + actor.Username + "@" + dt.NameOnly(actor.ID)
	}

	return actor.ID
}
