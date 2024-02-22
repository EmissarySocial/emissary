package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/rs/zerolog/log"
)

// AllowSend returns TRUE if this actorID is allowed to receive messages.
func (filter *RuleFilter) AllowSend(actorID string) bool {

	const location = "service.RuleFilter.AllowSend"

	log.Trace().Str("loc", location).Msg("Testing: " + actorID)

	// Guarantee that the actorID is not empty
	if actorID == "" {
		log.Trace().Str("loc", location).Msg("Ignore Empty actorID")
		return false
	}

	// We don't actually send messages to the public namespace
	if actorID == vocab.NamespaceActivityStreamsPublic {
		log.Trace().Str("loc", location).Msg("Ignore Public Namespace")
		return false
	}

	// If we don't have a cached value for this actor, then load it from the database.
	if filter.cache[actorID] == nil {

		allowedActions := filter.allowedActions()
		rules, err := filter.ruleService.QueryByActorAndActions(filter.userID, actorID, allowedActions...)

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error loading rules"))
			return false
		}

		filter.cache[actorID] = rules
	}

	for _, rule := range filter.cache[actorID] {
		if rule.IsDisallowSend(actorID) {
			log.Trace().Str("loc", location).Str("to", actorID).Msg("Disallowed by Rule")
			return false
		}
	}

	return true
}

// ChannelSend inspects the channel of recipients to see if they should receive messages or not.
func (filter *RuleFilter) ChannelSend(ch <-chan model.Follower) <-chan string {

	result := make(chan string)
	go func() {
		defer close(result)

		for follower := range ch {
			if filter.AllowSend(follower.Actor.ProfileURL) {
				log.Trace().Str("loc", "service.RuleFilter.ChannelSend").Str("actorID", follower.Actor.ProfileURL).Msg("Allowed")
				result <- follower.Actor.ProfileURL
			} else {
				log.Trace().Str("loc", "service.RuleFilter.ChannelSend").Str("actorID", follower.Actor.ProfileURL).Msg("Blocked")
			}
		}
	}()

	return result
}
