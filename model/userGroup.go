package model

import "github.com/benpate/data/journal"

// UserGroup represents a group of actors (users)
type UserGroup struct {
	UserGroupID      string
	KeyEncryptingKey string

	journal.Journal `json:"journal" bson:"jounal"`
}
