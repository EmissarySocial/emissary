package model

import (
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Rule represents many kinds of filters that are applied to messages before they are added into a User's inbox
type Rule struct {
	RuleID      primitive.ObjectID `json:"ruleId"      bson:"_id"`         // Unique identifier of this Rule
	UserID      primitive.ObjectID `json:"userId"      bson:"userId"`      // Unique identifier of the User who owns this Rule
	Type        string             `json:"type"        bson:"type"`        // Type of Rule (e.g. "ACTOR", "DOMAIN", "CONTENT")
	Action      string             `json:"action"      bson:"action"`      // Action to take when this rule is triggered (e.g. "BLOCK", "MUTE", "LABEL")
	Label       string             `json:"label"       bson:"label"`       // Human-friendly label to add to messages
	Trigger     string             `json:"trigger"     bson:"trigger"`     // Parameter for this rule type)
	Summary     string             `json:"summary"     bson:"summary"`     // Optional comment describing why this rule exists
	IsPublic    bool               `json:"isPublic"    bson:"isPublic"`    // If TRUE, this record is visible publicly
	Origin      OriginLink         `json:"origin"      bson:"origin"`      // Internal or External service where this rule originated (used for subscriptions)
	PublishDate int64              `json:"publishDate" bson:"publishDate"` // Unix timestamp when this rule was published to followers
	JSONLD      mapof.Any          `json:"jsonld"      bson:"jsonld"`      // JSON-LD data for this object

	journal.Journal `json:"-" bson:",inline"`
}

func NewRule() Rule {
	return Rule{
		RuleID:   primitive.NewObjectID(),
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
		"type",
		"action",
		"label",
		"trigger",
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

	// Folders are private, so only MagicRoleMyself is allowed
	if authorization.UserID == rule.UserID {
		return []string{MagicRoleMyself}
	}

	// Intentionally NOT allowing MagicRoleAnonymous, MagicRoleAuthenticated, or MagicRoleOwner
	return []string{}
}

/******************************************
 * ActivityStreams Methods
 ******************************************/

// GetJSONLD returns a map document that conforms to the ActivityStreams 2.0 spec.
// This map will still need to be marshalled into JSON
func (rule Rule) GetJSONLD() mapof.Any {
	return rule.JSONLD
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

func (rule Rule) GetRank() int64 {
	return rule.CreateDate
}

/******************************************
 * Other Data Accessors
 ******************************************/

func (rule Rule) PublishDateRCF3339() string {
	return time.Unix(rule.PublishDate, 0).Format(time.RFC3339)
}
