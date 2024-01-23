package model

import (
	"strings"

	"github.com/benpate/hannibal/streams"
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

	// Apply content filters here.  If the content matches the filter, then dissallow the document.
	if rule.Type == RuleTypeContent {

		if strings.Contains(document.Name(), rule.Trigger) {
			return true
		}

		if strings.Contains(document.Summary(), rule.Trigger) {
			return true
		}

		if strings.Contains(document.Content(), rule.Trigger) {
			return false
		}

		for tag := document.Tag(); tag.NotNil(); tag = tag.Next() {
			if strings.Contains(tag.Name(), rule.Trigger) {
				return false
			}
		}

		// Last, if the document contains a disallowed Object,
		// then it is disallowed, too.
		if object := document.Object(); object.NotNil() {
			if rule.IsDisallowed(&object) {
				return true
			}
		}
	}

	// All actions (except label) will disallow the document
	if rule.Action != RuleActionLabel {
		return true
	}

	// Label actions add a label to the document, but do not disallow it.
	// TODO: Add label actions.

	return false
}
