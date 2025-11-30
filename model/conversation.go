package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation represents a group of messages sent among several participants
type Conversation struct {
	ConversationID primitive.ObjectID         `bson:"_id"`          // Unique ID for this folder
	UserID         primitive.ObjectID         `bson:"userId"`       // ID of the User who owns this folder
	Participants   sliceof.Object[PersonLink] `bson:"participants"` // List of people who are participating in this conversation
	Name           string                     `bson:"name"`         // Name of the conversation
	Comment        string                     `bson:"comment"`      // User notes
	Icon           string                     `bson:"icon"`         // Icon of the folder
	StateID        string                     `bson:"stateId"`      // Current state of this conversation (UNREAD, READ, ARCHIVED)

	journal.Journal `json:"-" bson:"journal"`
}

// NewConversation returns a fully initialized Conversation object
func NewConversation() Conversation {
	return Conversation{
		ConversationID: primitive.NewObjectID(),
		StateID:        "UNREAD",
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (conversation Conversation) ID() string {
	return conversation.ConversationID.Hex()
}

/******************************************
 * FieldLister Interface
 ******************************************/

func (conversation Conversation) Fields() []string {
	return []string{
		"conversationId",
		"participants",
		"name",
		"notes",
		"icon",
		"stateId",
	}
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Conversation.
// It is part of the AccessLister interface
func (conversation *Conversation) State() string {
	return conversation.StateID
}

// IsAuthor returns TRUE if the provided UserID the author of this Conversation
// It is part of the AccessLister interface
func (conversation *Conversation) IsAuthor(authorID primitive.ObjectID) bool {
	return !authorID.IsZero() && authorID == conversation.UserID
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (conversation *Conversation) IsMyself(userID primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (conversation *Conversation) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(conversation.UserID, roleIDs...)
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (conversation *Conversation) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}
