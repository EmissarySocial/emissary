package model

import "github.com/benpate/data/journal"

// Actor represents an account or node on this server.
type Actor struct {
	ActorID  string `json:"actorId"  bson:"_id"`       // This is the internal ID for the actor.  It should not be available via the web service.
	Username string `json:"username" bson:"username"`  // This is the primary public identifier for the user.
	Password string `json:"password" bsnon:"password"` // This password should be encrypted with BCrypt.

	journal.Journal
}

// ID returns the unique identifier of this object, and fulfills part of the data.Object interface
func (actor *Actor) ID() string {
	return actor.ActorID
}
