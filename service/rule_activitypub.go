package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

/******************************************
 * ActivityPub Methods
 ******************************************/

func (service *Rule) ActivityPubActorURL(rule model.Rule) string {
	return service.host + "/@" + rule.UserID.Hex()
}

func (service *Rule) ActivityPubURL(rule model.Rule) string {
	return service.ActivityPubActorURL(rule) + "/pub/blocked/" + rule.RuleID.Hex()
}

// JSONLDGetter returns a new JSONLDGetter for the provided stream
func (service *Rule) JSONLDGetter(rule model.Rule) RuleJSONLDGetter {
	return NewRuleJSONLDGetter(service, rule)
}

func (service *Rule) Activity(rule model.Rule) streams.Document {
	return service.activityService.NewDocument(service.JSONLD(rule))
}

func (service *Rule) ActivityType(rule model.Rule) string {

	// Determine the ActivityPub type for this Rule
	switch rule.Action {

	case model.RuleActionBlock:
		return vocab.ActivityTypeBlock

	case model.RuleActionMute:
		return vocab.ActivityTypeIgnore

	case model.RuleActionLabel:
		return vocab.ActivityTypeFlag
	}

	// If we don't know the type, then return an empty string
	return ""
}

// JSONLD returns a JSON-LD representation of the provided Rule
func (service *Rule) JSONLD(rule model.Rule) mapof.Any {

	// Reset JSON-LD for the rule.  We're going to recalculate EVERYTHING.
	result := mapof.Any{
		vocab.PropertyID:        service.ActivityPubURL(rule),
		vocab.PropertyPublished: hannibal.TimeFormat(time.Unix(rule.PublishDate, 0)),
		vocab.PropertyType:      service.ActivityType(rule),
	}

	// Create the summary based on the type of Rule
	switch rule.Type {

	case model.RuleTypeActor:
		result[vocab.PropertyObject] = mapof.Any{
			vocab.PropertyType: vocab.ActorTypePerson,
			vocab.PropertyID:   rule.Trigger,
		}

	case model.RuleTypeContent:
		result[vocab.PropertyObject] = mapof.Any{
			vocab.PropertyType:    vocab.ObjectTypeNote,
			vocab.PropertyContent: rule.Trigger,
		}

	case model.RuleTypeDomain:
		result[vocab.PropertyObject] = mapof.Any{
			vocab.PropertyType: vocab.ActorTypeService,
			vocab.PropertyID:   rule.Trigger,
			vocab.PropertyURL:  rule.Trigger,
		}
	}

	// TODO: need additional grammar for extra fields
	// - selectbox field to describe WHY the rule was created
	// - comment field to describe WHY the rule was created
	// - refs to other people who have ALSO ruleed this person/domain/keyword?

	return result
}
