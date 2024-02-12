package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
)

// AllowSend returns TRUE if this actorID is allowed to receive messages.
func (filter *RuleFilter) AllowSend(actorID string) bool {

	// Guarantee that the actorID is not empty
	if actorID == "" {
		return false
	}

	// We don't actually send messages to the public namespace
	if actorID == vocab.NamespaceActivityStreamsPublic {
		return false
	}

	// If we don't have a cached value for this actor, then load it from the database.
	if filter.cache[actorID] == nil {

		allowedActions := filter.allowedActions()
		rules, err := filter.ruleService.QueryByActorAndActions(filter.userID, actorID, allowedActions...)

		if err != nil {
			derp.Report(derp.Wrap(err, "emissary.RuleFilter.FilterOne", "Error loading rules"))
			return false
		}

		filter.cache[actorID] = rules
	}

	return len(filter.cache[actorID]) == 0
}

// ChannelSend inspects the channel of recipients to see if they should receive messages or not.
func (filter *RuleFilter) ChannelSend(ch <-chan model.Follower) <-chan string {

	result := make(chan string)

	go func() {
		defer close(result)

		for follower := range ch {
			if filter.AllowSend(follower.Actor.ProfileURL) {
				result <- follower.Actor.ProfileURL
			}
		}
	}()

	return result
}
