package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/datetime"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
)

/******************************************
 * ActivityPub Methods
 ******************************************/

// ActivityPubActorURL returns the URL of the Actor that owns the provided Rule
func (service *Rule) ActivityPubActorURL(rule model.Rule) string {
	return service.host + "/@" + rule.UserID.Hex()
}

// ActivityPubURL returns the canonical URL of the provided Rule's published activity.
//
// NOTE: nothing serves this URL. Users stopped publishing a "/pub/blocked" collection when Rule
// federation became a Domain-only capability (D9), and the Domain has no publishing path yet, so the
// whole publish path is dormant. Whatever serves a Domain's published policy will own this URL space
// and must redefine this. See RULES.md D9.
func (service *Rule) ActivityPubURL(rule model.Rule) string {
	return service.ActivityPubActorURL(rule) + "/pub/blocked/" + rule.RuleID.Hex()
}

// Activity returns a Rule as a hannibal ActivityStreams Document
func (service *Rule) Activity(rule model.Rule) streams.Document {
	return streams.NewDocument(service.JSONLD(rule))
}

// ActivityType returns the ActivityPub type that a Rule federates as, or an empty string if it does
// not federate.
func (service *Rule) ActivityType(rule model.Rule) string {

	// LABEL is absent on purpose. Everywhere else in the Fediverse a `Flag` is a PRIVATE moderation
	// report to a server's moderators, so a personal label broadcast as a Flag reads to a conformant
	// receiver as an abuse report against the labelled Actor. Labels stay local until the Notice
	// store ships. See RULES.md R15.
	switch rule.Action {

	case model.RuleActionBlock:
		return vocab.ActivityTypeBlock

	case model.RuleActionMute:
		return vocab.ActivityTypeIgnore
	}

	// If we don't know the type, then return an empty string
	return ""
}

// publishedJSONLD recomposes the Rule's activity AS LAST PUBLISHED (P7-2): the live Rule with its
// Action replaced by PublishedAction, so a retraction embeds the activity type the wire actually
// saw -- never the live Action, which may have changed since. An empty PublishedAction (a row
// stamped before the field existed) falls back to the live Action, which is exactly the old
// behavior. Object-body drift from later Trigger edits is accepted: receivers key a retraction
// on the embedded activity's `id`, which never changes.
func (service *Rule) publishedJSONLD(rule model.Rule) mapof.Any {

	if rule.PublishedAction != "" {
		rule.Action = rule.PublishedAction
	}

	return service.JSONLD(rule)
}

// JSONLD returns a JSON-LD representation of the provided Rule
func (service *Rule) JSONLD(rule model.Rule) mapof.Any {

	// Reset JSON-LD for the rule.  We're going to recalculate EVERYTHING.
	result := mapof.Any{
		vocab.PropertyID:   service.ActivityPubURL(rule),
		vocab.PropertyType: service.ActivityType(rule),
	}

	// PublishDate is 0 for an unpublished Rule. Omit `published` in that case
	// instead of dating the rule to the Unix epoch.
	if published := datetime.FromUnixSeconds(rule.PublishDate); published != "" {
		result[vocab.PropertyPublished] = published
	}

	// Create the summary based on the type of Rule
	switch rule.Type {

	case model.RuleTypeActor:
		result[vocab.PropertyObject] = mapof.Any{
			vocab.PropertyType: vocab.ActorTypePerson,
			vocab.PropertyID:   rule.Trigger,
		}

	// RULE: P7-1 -- a TAG rule federates as a Hashtag object, extending the one wire grammar
	// (activity = action, object = trigger kind). `name` carries the Mastodon-convention form
	// ("#token", ToToken-normalized on both publish and ingest). No `href` until a canonical
	// tag page exists -- receivers key on `name` regardless, because href is per-instance.
	case model.RuleTypeTag:
		result[vocab.PropertyObject] = mapof.Any{
			vocab.PropertyType: vocab.LinkTypeHashtag,
			vocab.PropertyName: "#" + model.ToToken(rule.Trigger),
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
