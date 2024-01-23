package model

import "github.com/benpate/domain"

// ActorSummary is a record returned by the ActivityStreams directory
type ActorSummary struct {
	ID       string `bson:"id"`
	Type     string `bson:"type"`
	Name     string `bson:"name"`
	Icon     string `bson:"icon"`
	Username string `bson:"preferredUsername"`
}

func (actor ActorSummary) UsernameOrID() string {
	if actor.Username != "" {
		return "@" + actor.Username + "@" + domain.NameOnly(actor.ID)
	}

	return actor.ID
}
