package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Collection represents a group of messages sent among several participants
type Collection struct {
	CollectionID primitive.ObjectID `bson:"_id"`    // Unique ID for this folder
	UserID       primitive.ObjectID `bson:"userId"` // ID of the User who owns this folder
	Name         string             `bson:"name"`   // Name of the collection
	To           sliceof.String     `bson:"to"`     // List of people who are participating in this collection
	Cc           sliceof.String     `bson:"cc"`     // List of people who are participating in this collection

	journal.Journal `json:"-" bson:",inline"`
}

// NewCollection returns a fully initialized Collection object
func NewCollection() Collection {
	return Collection{
		CollectionID: primitive.NewObjectID(),
		To:           sliceof.NewString(),
		Cc:           sliceof.NewString(),
	}
}

// HasParticipant returns TRUE if the provided actor is allowed to read this Collection
func (collection *Collection) HasParticipant(actorID string) bool {

	// RULE: A Collection addressed to the public is readable by anyone (incl. an empty/unauthenticated actor)
	if collection.isPublic() {
		return true
	}

	// Otherwise, the actor must be named in the "to" or "cc" participant lists
	if actorID == "" {
		return false
	}

	return collection.To.Contains(actorID) || collection.Cc.Contains(actorID)
}

// isPublic returns TRUE if this Collection is addressed to the public in its "to" or "cc" lists
func (collection *Collection) isPublic() bool {
	return isPublicRecipient(collection.To) || isPublicRecipient(collection.Cc)
}

// isPublicRecipient returns TRUE if any entry in the list is an ActivityPub public-addressing token
func isPublicRecipient(recipients sliceof.String) bool {

	for _, recipient := range recipients {
		switch recipient {
		case vocab.NamespaceActivityStreamsPublic, vocab.NamespaceASPublic, vocab.NamespacePublic:
			return true
		}
	}

	return false
}

/******************************************
 * data.Object Interface
 ******************************************/

func (collection Collection) ID() string {
	return collection.CollectionID.Hex()
}

/******************************************
 * FieldLister Interface
 ******************************************/

func (collection Collection) Fields() []string {
	return []string{
		"collectionId",
		"to",
		"cc",
		"name",
	}
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Collection.
// It is part of the AccessLister interface
func (collection *Collection) State() string {
	return "DEFAULT"
}

// IsAuthor returns TRUE if the provided UserID the author of this Collection
// It is part of the AccessLister interface
func (collection *Collection) IsAuthor(authorID primitive.ObjectID) bool {
	return !authorID.IsZero() && authorID == collection.UserID
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (collection *Collection) IsMyself(userID primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (collection *Collection) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(collection.UserID, roleIDs...)
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (collection *Collection) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}
