package model

import (
	"strings"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleSummary is a trimmed down subset of the Rule object, which is used when
// executing rules on a piece of content
type RuleSummary struct {
	RuleID   primitive.ObjectID `bson:"_id"`
	Type     string             `bson:"type"`
	Action   string             `bson:"action"`
	Behavior string             `bson:"behavior"`
	Trigger  string             `bson:"trigger"`
}

// RuleSummaryFields returns a list of fields that should be queried from the
// database when populating a RuleSummary object or collection.
func RuleSummaryFields() []string {
	return []string{
		"_id",
		"type",
		"action",
		"behavior",
		"trigger",
	}
}

func (rule RuleSummary) Fields() []string {
	return RuleSummaryFields()
}

// IsAllowed returns TRUE if the document should be allowed based on
// this rule.  (i.e. the document DOES NOT match the rule)
func (rule RuleSummary) IsAllowed(document *streams.Document) bool {
	return !rule.IsDisallowed(document)
}

// IsDisallowed returns TRUE if the document SHOULD NOT BE allowed based on
// this rule.  (i.e. the document MATCHES the rule)
func (rule RuleSummary) IsDisallowed(document *streams.Document) bool {

	// Apply content filters here.
	if rule.Type == RuleTypeContent {

		// If the document does not match the content filter, then it is allowed.
		if !rule.matchesContent(document) {
			return false
		}
	}

	// All actions (except label) will disallow the document
	if rule.Action != RuleActionLabel {
		return true
	}

	// Label actions add a label to the document, but do not disallow it.
	// TODO: Add label actions here.

	return false
}

func (rule RuleSummary) matchesContent(document *streams.Document) bool {

	// RULE: Only applies to Content rules.  All others are not blocked
	if rule.Type != RuleTypeContent {
		return false
	}

	// RULE: Do not block "Block" activities.  Otherwise, they are un-updatable.
	if document.Type() == vocab.ActivityTypeBlock {
		return false
	}

	// RULE: Try to match NAME against the trigger
	if strings.Contains(document.Name(), rule.Trigger) {
		log.Trace().Msg("disallowed because of name")
		return true
	}

	// RULE: Try to match SUMMARY against the trigger
	if strings.Contains(document.Summary(), rule.Trigger) {
		log.Trace().Msg("disallowed because of summary")
		return true
	}

	// RULE: Try to match CONTENT against the trigger
	if strings.Contains(document.Content(), rule.Trigger) {
		log.Trace().Msg("disallowed because of content")
		return true
	}

	// RULE: Try to match TAGS against the trigger
	for tag := document.Tag(); tag.NotNil(); tag = tag.Next() {
		if tag.Name() == rule.Trigger {
			log.Trace().Msg("disallowed because of tag" + tag.Name())
			return true
		}
	}

	// Last, if the document contains a disallowed Object,
	// then it is disallowed, too.
	if object := document.Object(); object.NotNil() {
		if rule.IsDisallowed(&object) {
			log.Trace().Msg("... disallowed because of object")
			return true
		}
	}

	return false
}
