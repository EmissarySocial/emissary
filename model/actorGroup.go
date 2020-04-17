package model

import "github.com/benpate/data/journal"

// ActorGroup represents a group of actors (users)
type ActorGroup struct {
	ActorGroupID     string
	KeyEncryptingKey string

	journal.Journal `json:"journal" bson:"jounal"`
}
