package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

/******************************************
 * Filters
 ******************************************/

func (service *Rule) FilterFollower(follower *model.Follower) error {

	// RULE: Rules ONLY work on ActivityPub followers
	if follower.Method != model.FollowMethodActivityPub {
		return nil
	}

	// Get a list of all rules for this User
	activeRules, err := service.QueryActiveByUser(follower.ParentID)

	if err != nil {
		return derp.Wrap(err, "service.Rule.FilterFollower", "Error loading rules for user", follower.ParentID)
	}

	// Try each rule. If "BLOCK", then do not allow the follower
	for _, rule := range activeRules {
		if rule.FilterByActor(follower.Actor.ProfileURL) {
			return derp.NewValidationError("Actor blocked")
		}
	}

	// No block means that this follower is allowed
	return nil
}

func (service *Rule) FilterMention(mention *model.Mention) error {

	// Get a list of all rules for this User
	activeRules, err := service.QueryActiveByUser(mention.ObjectID)

	if err != nil {
		return derp.Wrap(err, "service.Rule.FilterFollower", "Error loading rules for user", mention.ObjectID)
	}

	// Try each rule.  If "BLOCK" or "MUTE", then do not allow the mention
	for _, rule := range activeRules {
		if rule.FilterByActors(mention.Origin.URL, mention.Author.ProfileURL) {
			return derp.NewValidationError("Actor blocked")
		}
	}

	// No block means that this follower is allowed
	return nil
}

func (service *Rule) FilterStreamResponse(stream *model.Stream, actors ...string) error {

	// Get a list of all rules for this User
	activeRules, err := service.QueryActiveByUser(stream.ParentID)

	if err != nil {
		return derp.Wrap(err, "service.Rule.FilterFollower", "Error loading rules for user", stream.ParentID)
	}

	// Try each rule.  If "BLOCK" or "MUTE", then do not allow the mention
	for _, rule := range activeRules {
		if rule.FilterByActors(actors...) {
			return derp.NewValidationError("Actor blocked")
		}
	}

	// No block means that this follower is allowed
	return nil
}

func (service *Rule) FilterMessage(message *model.Message) error {

	// Get a list of all rules for this User
	activeRules, err := service.QueryActiveByUser(message.UserID)

	if err != nil {
		return derp.Wrap(err, "service.Rule.filterMessage", "Error loading rules for user", message.UserID)
	}

	// Try to execute each rule
	for _, rule := range activeRules {
		if rule.FilterByActorAndContent(message.Origin.URL, "", "", "") { // message.Label, message.Summary, message.ContentHTML) {
			return derp.NewValidationError("Actor blocked")
		}
	}

	return nil
}
