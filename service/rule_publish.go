package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

/******************************************
 * Publishing Methods
 ******************************************/

// publish marks the Rule as published, and sends "Create" activities to all ActivityPub followers
func (service *Rule) publish(rule model.Rule) error {

	// Get an ActivityPub Actor for this User
	actor, err := service.userService.ActivityPubActor(rule.UserID, true)

	if err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error loading User", rule)
	}

	// Publish this Rule to the User's outbox
	if err := service.outboxService.Publish(&actor, model.FollowerTypeUser, rule.UserID, service.JSONLD(rule)); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error publishing Rule", rule)
	}

	return nil
}

// unpublish marks the Rule as unpublished and sends "Undo" activities to all ActivityPub followers
func (service *Rule) unpublish(rule model.Rule) error {

	// Get an ActivityPub Actor for this User
	actor, err := service.userService.ActivityPubActor(rule.UserID, true)

	if err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error loading User", rule)
	}

	// UnPublish this Rule from the User's outbox
	if err := service.outboxService.UnPublish(&actor, model.FollowerTypeUser, rule.UserID, service.ActivityPubURL(rule)); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error publishing Rule", rule)
	}

	return nil
}

func (service *Rule) republish(rule model.Rule) error {

	// Get an ActivityPub Actor for this User
	actor, err := service.userService.ActivityPubActor(rule.UserID, true)

	if err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error loading User", rule)
	}

	// UnPublish the original Rule from the User's outbox
	if err := service.outboxService.UnPublish(&actor, model.FollowerTypeUser, rule.UserID, service.ActivityPubURL(rule)); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error publishing Rule", rule)
	}

	// Publish the updated Rule to the User's outbox
	if err := service.outboxService.Publish(&actor, model.FollowerTypeUser, rule.UserID, service.JSONLD(rule)); err != nil {
		return derp.Wrap(err, "service.Rule.Save", "Error publishing Rule", rule)
	}

	return nil
}
