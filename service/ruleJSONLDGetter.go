package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
)

// RuleJSONLDGetter wraps the Rule service and a model.Rule to provide a JSONLDGetter interface
type RuleJSONLDGetter struct {
	service *Rule
	rule    model.Rule
}

// NewRuleJSONLDGetter returns a fully initialized RuleJSONLDGetter
func NewRuleJSONLDGetter(service *Rule, rule model.Rule) RuleJSONLDGetter {
	return RuleJSONLDGetter{
		service: service,
		rule:    rule,
	}
}

// GetJSONLD returns a JSON-LD representation of the wrapped Rule
func (getter RuleJSONLDGetter) GetJSONLD() mapof.Any {
	return getter.service.JSONLD(getter.rule)
}

// Created returns the creation date of the wrapped Rule
func (getter RuleJSONLDGetter) Created() int64 {
	return getter.rule.CreateDate
}
