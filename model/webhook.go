package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Webhook defines an outbound webhook that can be triggered by events in the system
type Webhook struct {
	WebhookID       primitive.ObjectID `bson:"_id"`
	Events          sliceof.String     `bson:"events"`
	Label           string             `bson:"label"`
	TargetURL       string             `bson:"targetUrl"`
	journal.Journal `json:"-" bson:",inline"`
}

// NewWebhook returns a fully initialized Webhook object
func NewWebhook() Webhook {
	return Webhook{
		WebhookID: primitive.NewObjectID(),
		Events:    sliceof.NewString(),
	}
}

func WebhookFields() []string {
	return []string{"_id", "events", "label", "targetUrl"}
}

func (userSummary Webhook) Fields() []string {
	return WebhookFields()
}

// ID returns the unique identifier for this Webhook, and is required to implement the data.Object interface
func (webhook Webhook) ID() string {
	return webhook.WebhookID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Webhook.
// It is part of the AccessLister interface
func (webhook *Webhook) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Webhook
// It is part of the AccessLister interface
func (webhook *Webhook) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (webhook *Webhook) IsMyself(userID primitive.ObjectID) bool {
	return false
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (webhook *Webhook) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(primitive.NilObjectID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (webhook *Webhook) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}
