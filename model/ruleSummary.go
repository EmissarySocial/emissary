package model

import (
	"net/url"
	"strings"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleSummary is a trimmed down subset of the Rule object, which is used when
// executing rules on a piece of content
type RuleSummary struct {
	RuleID         primitive.ObjectID `bson:"_id"`
	Type           string             `bson:"type"`
	Action         string             `bson:"action"`
	Behavior       string             `bson:"behavior"`
	Trigger        string             `bson:"trigger"`
	Label          string             `bson:"label"`
	FollowingLabel string             `bson:"followingLabel"`
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
		"label",
		"followingLabel",
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

	switch rule.Type {

	case RuleTypeActor:

		if document.Actor().ID() != rule.Trigger {
			return false
		}

	case RuleTypeDomain:
		if domain, err := url.Parse(document.Actor().ID()); err == nil {
			if !strings.HasSuffix(domain.Hostname(), rule.Trigger) {
				return false
			}
		}

	case RuleTypeContent:

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
	document.Append(vocab.PropertyTag, map[string]any{
		vocab.PropertyHref:    "/@me/inbox/rule-edit?ruleId=" + rule.RuleID.Hex(),
		vocab.PropertyRel:     TagRelationRule,
		vocab.PropertyName:    rule.FollowingLabel,
		vocab.PropertyContent: rule.Label,
	})

	return false
}

func (rule RuleSummary) IsDisallowSend(recipient string) bool {

	switch rule.Type {

	case RuleTypeActor:
		return recipient == rule.Trigger

	case RuleTypeDomain:
		if domain, err := url.Parse(recipient); err == nil {
			return strings.HasSuffix(domain.Hostname(), rule.Trigger)
		}
	}

	return false
}

func (rule RuleSummary) matchesContent(document *streams.Document) bool {

	ruleTriggerLowerCase := strings.ToLower(rule.Trigger)

	// RULE: Only applies to Content rules.  All others are not blocked
	if rule.Type != RuleTypeContent {
		return false
	}

	// RULE: Do not block "Block" activities.  Otherwise, they are un-updatable.
	if document.Type() == vocab.ActivityTypeBlock {
		return false
	}

	// RULE: Try to match NAME against the trigger
	if strings.Contains(strings.ToLower(document.Name()), ruleTriggerLowerCase) {
		log.Trace().Msg("disallowed because of name")
		return true
	}

	// RULE: Try to match SUMMARY against the trigger
	if strings.Contains(strings.ToLower(document.Summary()), ruleTriggerLowerCase) {
		log.Trace().Msg("disallowed because of summary")
		return true
	}

	// RULE: Try to match CONTENT against the trigger
	if strings.Contains(strings.ToLower(document.Content()), ruleTriggerLowerCase) {
		log.Trace().Msg("disallowed because of content")
		return true
	}

	// RULE: Try to match TAGS against the trigger
	for tag := document.Tag(); tag.NotNil(); tag = tag.Next() {
		if strings.ToLower(tag.Name()) == ruleTriggerLowerCase {
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
