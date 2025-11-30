package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Rule represents many kinds of filters that are applied to messages before they are added into a User's inbox
type Rule struct {
	RuleID         primitive.ObjectID `bson:"_id"`            // Unique identifier of this Rule
	UserID         primitive.ObjectID `bson:"userId"`         // Unique identifier of the User who owns this Rule
	FollowingID    primitive.ObjectID `bson:"followingId"`    // Unique identifier of the Following record that created this Rule.  If Zero, then this rule was created by the user.
	FollowingLabel string             `bson:"followingLabel"` // Label of the Following record that created this Rule.
	Type           string             `bson:"type"`           // Type of Rule (e.g. "ACTOR", "DOMAIN", "CONTENT")
	Action         string             `bson:"action"`         // Action to take when this rule is triggered (e.g. "BLOCK", "MUTE", "LABEL")
	Label          string             `bson:"label"`          // Human-friendly label to add to messages
	Trigger        string             `bson:"trigger"`        // Parameter for this rule type)
	Summary        string             `bson:"summary"`        // Optional comment describing why this rule exists
	IsPublic       bool               `bson:"isPublic"`       // If TRUE, this record is visible publicly
	PublishDate    int64              `bson:"publishDate"`    // Unix timestamp when this rule was published to followers

	journal.Journal `json:"-" bson:",inline"`
}

func NewRule() Rule {
	return Rule{
		RuleID:   primitive.NewObjectID(),
		Type:     RuleTypeActor,
		Action:   RuleActionMute,
		IsPublic: false,
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (rule Rule) ID() string {
	return rule.RuleID.Hex()
}

func (rule Rule) Fields() []string {
	return []string{
		"_id",
		"userId",
		"followingId",
		"type",
		"action",
		"label",
		"trigger",
		"summary",
		"isPublic",
	}
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Rule.
// It is part of the AccessLister interface
func (rule *Rule) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided RuleID the author of this Rule
// It is part of the AccessLister interface
func (rule *Rule) IsAuthor(authorID primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided RuleID
// It is part of the AccessLister interface
func (rule *Rule) IsMyself(userID primitive.ObjectID) bool {
	return !userID.IsZero() && userID == rule.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (rule *Rule) RolesToGroupIDs(roleIDs ...string) Permissions {
	return defaultRolesToGroupIDs(rule.UserID, roleIDs...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles
// It is part of the AccessLister interface
func (rule *Rule) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Mastodon API Methods
 ******************************************/

func (rule Rule) Toot() object.Relationship {
	return object.Relationship{
		ID:       rule.Trigger,
		Blocking: !rule.IsDeleted(),
	}
}

// GetRank returns the "Rank" of this object, which is its CreateDate
func (rule Rule) GetRank() int64 {
	return rule.CreateDate
}

// Origin returns a string that identifies the origin of this Rule. (DOMAIN, REMOTE, or USER)
func (rule Rule) Origin() string {

	if rule.OriginAdmin() {
		return RuleOriginAdmin
	}

	if rule.OriginRemote() {
		return RuleOriginRemote
	}

	return RuleOriginUser
}

// OriginAdmin returns TRUE if this Rule was created by a Domain administrator.
func (rule Rule) OriginAdmin() bool {
	return rule.UserID.IsZero()
}

// OriginRemote returns TRUE if this Rule was created by a Following record.
func (rule Rule) OriginRemote() bool {
	return !rule.OriginUser()
}

// OriginUser returns TRUE if this Rule was created by the User.
func (rule Rule) OriginUser() bool {
	return rule.FollowingID.IsZero()
}
