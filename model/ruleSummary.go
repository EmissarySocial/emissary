package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleSummary is a trimmed down subset of the Rule object, which is used when
// executing rules on a piece of content
type RuleSummary struct {
	RuleID         primitive.ObjectID `bson:"_id"`
	UserID         primitive.ObjectID `bson:"userId"` // Owner; zero => ADMIN (domain) tier. REQUIRED for tier attribution.
	Type           string             `bson:"type"`
	Action         string             `bson:"action"`
	Trigger        string             `bson:"trigger"`
	MatchKey       string             `bson:"matchKey"` // Derived key; a document matches this rule iff its key set contains this value.
	Label          string             `bson:"label"`
	FollowingLabel string             `bson:"followingLabel"`
	ExpireDate     int64              `bson:"expireDate"` // 0 = never; an expired rule is skipped by the engine.
}

// RuleSummaryFields returns a list of fields that should be queried from the
// database when populating a RuleSummary object or collection.
//
// IMPORTANT: `userId` is load-bearing. If it is dropped, every rule projects a zero UserID and the
// engine reads the entire rule set as ADMIN-tier. TestRuleSummaryFields pins this against the struct.
func RuleSummaryFields() []string {
	return []string{
		"_id",
		"userId",
		"type",
		"action",
		"trigger",
		"matchKey",
		"label",
		"followingLabel",
		"expireDate",
	}
}

// Fields returns the database fields required to populate a RuleSummary
func (rule RuleSummary) Fields() []string {
	return RuleSummaryFields()
}
