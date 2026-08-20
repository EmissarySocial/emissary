package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleSchema returns the rosetta schema that describes a Rule
func RuleSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			// NOTE: `followingId` is deliberately ABSENT. It is written only by the import path
			// (service/rule_import.go); exposing it here would let a form post forge a Rule's origin
			// -- flipping Origin() to REMOTE and steering the IsPublic clamp and dedup merge. See D9.
			"ruleId":         schema.String{Required: true, Format: "objectId"},
			"userId":         schema.String{Required: true, Format: "objectId"},
			"followingLabel": schema.String{Format: "text", MaxLength: 64},
			"type":           schema.String{Required: true, Enum: []string{RuleTypeDomain, RuleTypeActor, RuleTypeTag}},
			"action":         schema.String{Required: true, Enum: []string{RuleActionBlock, RuleActionMute, RuleActionLabel}},
			// RULE: A LABEL rule with no label text matches documents but annotates nothing (LabelSet
			// skips empty labels), so it LOOKS active while doing nothing. Refuse it at validation
			// instead of saving it inert -- the same posture Save takes toward unresolvable Triggers.
			"label":       schema.String{Format: "text", MaxLength: 64, RequiredIf: "action is LABEL"},
			"trigger":     schema.String{MaxLength: 256, Required: true},
			"summary":     schema.String{Format: "text", MaxLength: 256},
			"reasonCode":  schema.String{MaxLength: 64},
			"isPublic":    schema.Boolean{},
			"publishDate": schema.Integer{BitSize: 64},
			"expireDate":  schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (rule *Rule) GetPointer(name string) (any, bool) {

	switch name {

	case "isPublic":
		return &rule.IsPublic, true

	case "publishDate":
		return &rule.PublishDate, true

	case "expireDate":
		return &rule.ExpireDate, true

	case "type":
		return &rule.Type, true

	case "followingLabel":
		return &rule.FollowingLabel, true

	case "action":
		return &rule.Action, true

	case "label":
		return &rule.Label, true

	case "trigger":
		return &rule.Trigger, true

	case "summary":
		return &rule.Summary, true

	case "reasonCode":
		return &rule.ReasonCode, true
	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (rule *Rule) GetStringOK(name string) (string, bool) {

	switch name {

	case "ruleId":
		return rule.RuleID.Hex(), true

	case "userId":
		return rule.UserID.Hex(), true

	case "followingId":
		return rule.FollowingID.Hex(), true

	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
func (rule *Rule) SetString(name string, value string) bool {

	switch name {

	case "ruleId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			rule.RuleID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			rule.UserID = objectID
			return true
		}

		// NOTE: no "followingId" case -- it is import-only and must not be settable from a form. See
		// RuleSchema and D9. Reading it back (GetStringOK) stays available for templates.
	}

	return false
}
