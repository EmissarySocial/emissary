package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Rule represents many kinds of filters that are applied to messages before they are added into a User's inbox
type Rule struct {
	RuleID         primitive.ObjectID `json:"ruleId"         bson:"_id"`            // Unique identifier of this Rule
	UserID         primitive.ObjectID `json:"userId"         bson:"userId"`         // Unique identifier of the User who owns this Rule
	FollowingID    primitive.ObjectID `json:"followingId"    bson:"followingId"`    // Unique identifier of the Following record that created this Rule.  If Zero, then this rule was created by the user.
	FollowingLabel string             `json:"followingLabel" bson:"followingLabel"` // Label of the Following record that created this Rule.
	Type           string             `json:"type"           bson:"type"`           // Type of Rule (e.g. "ACTOR", "DOMAIN", "CONTENT")
	Action         string             `json:"action"         bson:"action"`         // Action to take when this rule is triggered (e.g. "BLOCK", "MUTE", "LABEL")
	Label          string             `json:"label"          bson:"label"`          // Human-friendly label to add to messages
	Trigger        string             `json:"trigger"        bson:"trigger"`        // Parameter for this rule type)
	Summary        string             `json:"summary"        bson:"summary"`        // Optional comment describing why this rule exists
	IsPublic       bool               `json:"isPublic"       bson:"isPublic"`       // If TRUE, this record is visible publicly
	PublishDate    int64              `json:"publishDate"    bson:"publishDate"`    // Unix timestamp when this rule was published to followers

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
 * RoleStateEnumerator Interface
 ******************************************/

// State returns the current state of this object.
// For users, there is no state, so it returns ""
func (rule Rule) State() string {
	return ""
}

// Roles returns a list of all roles that match the provided authorization.
// Since Rule records should only be accessible by the rule owner, this
// function only returns MagicRoleMyself if applicable.  Others (like Anonymous
// and Authenticated) should never be allowed on an Rule record, so they
// are not returned.
func (rule Rule) Roles(authorization *Authorization) []string {

	// Rules are private, so only MagicRoleMyself is allowed
	if authorization.UserID == rule.UserID {
		return []string{MagicRoleMyself}
	}

	// Intentionally NOT allowing MagicRoleAnonymous, MagicRoleAuthenticated, or MagicRoleOwner
	return []string{}
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
