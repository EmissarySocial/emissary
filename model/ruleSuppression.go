package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleSuppression is the explicit don't-re-import record (P7-3). Under D16's hard delete, a
// deleted Rule leaves no tombstone -- so when an imported moderation entry is deleted locally,
// this record is what keeps the provider's next backfill from resurrecting it. A suppression
// names exactly one remote entry: a provider that retracts and re-publishes the same block under
// a NEW id is making a fresh assertion, and a fresh assertion imports again.
type RuleSuppression struct {
	RuleSuppressionID primitive.ObjectID `bson:"_id"`         // Unique identifier of this RuleSuppression
	UserID            primitive.ObjectID `bson:"userId"`      // Owner tier of the deleted Rule; zero => ADMIN/domain tier (the only subscriber, per D9)
	FollowingID       primitive.ObjectID `bson:"followingId"` // The provider subscription the suppressed entry arrived through
	RemoteID          string             `bson:"remoteId"`    // Canonical id (URL) of the provider's moderation entry that must not re-import

	journal.Journal `bson:",inline"`
}

// NewRuleSuppression returns a fully initialized RuleSuppression object
func NewRuleSuppression() RuleSuppression {
	return RuleSuppression{
		RuleSuppressionID: primitive.NewObjectID(),
	}
}

// ID returns the unique identifier of this RuleSuppression
// This is a part of the data.Object interface
func (suppression RuleSuppression) ID() string {
	return suppression.RuleSuppressionID.Hex()
}
