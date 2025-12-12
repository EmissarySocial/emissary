package model

import (
	dt "github.com/benpate/domain"
)

// ActorSummary is a record returned by the ActivityStream directory
type ActorSummary struct {
	ID          string `json:"actorId"     bson:"id"`
	Type        string `json:"type"        bson:"type"`
	Name        string `json:"name"        bson:"name"`
	Icon        string `json:"icon"        bson:"icon"`
	Username    string `json:"username"    bson:"username"`
	KeyPackages string `json:"keyPackages" bson:"keyPackages"`
}

// UsernameOrID returns the best identifier we can find for an Actor:
// either the Actor' username, if it exists, or the Actor's ID
func (actor ActorSummary) UsernameOrID() string {
	if actor.Username != "" {
		return "@" + actor.Username + "@" + dt.NameOnly(actor.ID)
	}

	return actor.ID
}
