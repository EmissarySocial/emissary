package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func RuleSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"ruleId":      schema.String{Required: true, Format: "objectId"},
			"userId":      schema.String{Required: true, Format: "objectId"},
			"type":        schema.String{Required: true, Enum: []string{RuleTypeDomain, RuleTypeActor, RuleTypeContent}},
			"action":      schema.String{Required: true, Enum: []string{RuleActionRule, RuleActionMute, RuleActionLabel}},
			"label":       schema.String{},
			"trigger":     schema.String{Required: true},
			"comment":     schema.String{},
			"origin":      OriginLinkSchema(),
			"isActive":    schema.Boolean{},
			"isPublic":    schema.Boolean{},
			"publishDate": schema.Integer{BitSize: 64},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (rule *Rule) GetPointer(name string) (any, bool) {

	switch name {

	case "origin":
		return &rule.Origin, true

	case "isPublic":
		return &rule.IsPublic, true

	case "isActive":
		return &rule.IsActive, true

	case "publishDate":
		return &rule.PublishDate, true

	case "type":
		return &rule.Type, true

	case "action":
		return &rule.Action, true

	case "label":
		return &rule.Label, true

	case "trigger":
		return &rule.Trigger, true

	case "comment":
		return &rule.Comment, true
	}

	return nil, false
}

func (rule *Rule) GetStringOK(name string) (string, bool) {

	switch name {

	case "ruleId":
		return rule.RuleID.Hex(), true

	case "userId":
		return rule.UserID.Hex(), true

	}

	return "", false
}

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
	}

	return false
}
