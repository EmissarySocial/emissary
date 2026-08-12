package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Collection represents a group of messages sent among several participants
type Collection struct {
	CollectionID   primitive.ObjectID `bson:"_id"`                      // Unique ID for this folder
	UserID         primitive.ObjectID `bson:"userId"`                   // ID of the User who owns this folder
	ParentID       primitive.ObjectID `bson:"parentId,omitempty"`       // ID of the object that owns this collection
	ParentType     string             `bson:"parentType,omitempty"`     // Type of the object that owns this collection
	CollectionType string             `bson:"collectionType,omitempty"` // Type of collection (Context, Replies, etc.)
	Read           sliceof.String     `bson:"read"`                     // List of people who are participating in this collection
	Write          sliceof.String     `bson:"write"`                    // List of people who are participating in this collection
	TotalItems     int                `bson:"totalItems"`               // Total number of items in this collection

	journal.Journal `json:"-" bson:",inline"`
}

// NewCollection returns a fully initialized Collection object
func NewCollection() Collection {
	return Collection{
		CollectionID: primitive.NewObjectID(),
		Read:         sliceof.NewString(),
		Write:        sliceof.NewString(),
	}
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
		"_id",
		"userId",
		"parentId",
		"parentType",
		"collectionType",
		"read",
		"write",
		"totalItems",
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

/******************************************
 * Other Permissions
 ******************************************/

// IsReadable returns TRUE if the provided actor is allowed to read this Collection
func (collection Collection) IsReadable(actorID string) bool {
	return collection.Read.ContainsAny(actorID, vocab.NamespacePublic)
}

// NotReadable returns TRUE if the provided actor is NOT allowed to read this Collection
func (collection Collection) NotReadable(actorID string) bool {
	return !collection.IsReadable(actorID)
}

// IsWritable returns TRUE if the provided actor is allowed to write to this Collection
func (collection Collection) IsWritable(actorID string) bool {
	return collection.Write.ContainsAny(actorID, vocab.NamespacePublic)
}

// NotWritable returns TRUE if the provided actor is NOT allowed to write to this Collection
func (collection Collection) NotWritable(actorID string) bool {
	return !collection.IsWritable(actorID)
}
