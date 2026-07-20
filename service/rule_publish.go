package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
)

/******************************************
 * Publishing Methods
 ******************************************/

// publish marks the Rule as published, and sends "Create" activities to all ActivityPub followers
func (service *Rule) publish(session data.Session, rule model.Rule) error {

	const location = "service.Rule.publish"

	// Publish this Rule to the User's outbox
	if err := service.outboxService.Publish(session, model.FollowerTypeUser, rule.UserID, service.Activity(rule), model.NewAnonymousPermissions()); err != nil {
		return derp.Wrap(err, location, "Publishing Rule", rule)
	}

	return nil
}

// unpublish marks the Rule as unpublished and sends "Undo" activities to all ActivityPub followers
func (service *Rule) unpublish(session data.Session, rule model.Rule) error {

	const location = "service.Rule.unpublish"

	// UnPublish this Rule from the User's outbox. A Block is a first-class ACTIVITY, so it is
	// UNDONE (embedding the original Block inline) — not deleted as an object. The embedded
	// original is recomposed from PublishedAction (P7-2): the Undo must name what the wire last
	// saw, not the live Action. See COLLECTIONS-REDESIGN.md D7.
	if err := service.outboxService.UndoActivity(session, model.FollowerTypeUser, rule.UserID, service.publishedJSONLD(rule), model.NewAnonymousPermissions()); err != nil {
		return derp.Wrap(err, location, "Retracting Rule", rule)
	}

	return nil
}

// republish retracts a Rule's previous activity and publishes its updated one
func (service *Rule) republish(session data.Session, rule model.Rule) error {

	const location = "service.Rule.republish"

	// UnPublish the original Rule from the User's outbox (Undo the Block activity — see D7).
	// The embedded original is recomposed from PublishedAction (P7-2), so an Action change
	// retracts what the wire last saw before publishing the new shape.
	if err := service.outboxService.UndoActivity(session, model.FollowerTypeUser, rule.UserID, service.publishedJSONLD(rule), model.NewAnonymousPermissions()); err != nil {
		return derp.Wrap(err, location, "Retracting previous Rule", rule)
	}

	// Publish the updated Rule to the User's outbox
	if err := service.outboxService.Publish(session, model.FollowerTypeUser, rule.UserID, service.Activity(rule), model.NewAnonymousPermissions()); err != nil {
		return derp.Wrap(err, location, "Publishing Rule", rule)
	}

	return nil
}
