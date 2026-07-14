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

	const location = "service.Rule.Save"

	// Publish this Rule to the User's outbox
	if err := service.outboxService.Publish(session, model.FollowerTypeUser, rule.UserID, service.Activity(rule), model.NewAnonymousPermissions()); err != nil {
		return derp.Wrap(err, location, "Error publishing Rule", rule)
	}

	return nil
}

// unpublish marks the Rule as unpublished and sends "Undo" activities to all ActivityPub followers
func (service *Rule) unpublish(session data.Session, rule model.Rule) error {

	const location = "service.Rule.unpublish"

	// UnPublish this Rule from the User's outbox. A Block is a first-class ACTIVITY, so it is
	// UNDONE (embedding the original Block inline, looked up by its own URL) — not deleted as an
	// object. See COLLECTIONS-REDESIGN.md D7.
	if err := service.outboxService.UndoActivity(session, model.FollowerTypeUser, rule.UserID, service.JSONLD(rule), model.NewAnonymousPermissions()); err != nil {
		return derp.Wrap(err, location, "Error publishing Rule", rule)
	}

	return nil
}

func (service *Rule) republish(session data.Session, rule model.Rule) error {

	const location = "service.Rule.republish"

	// UnPublish the original Rule from the User's outbox (Undo the Block activity — see D7).
	if err := service.outboxService.UndoActivity(session, model.FollowerTypeUser, rule.UserID, service.JSONLD(rule), model.NewAnonymousPermissions()); err != nil {
		return derp.Wrap(err, location, "Error publishing Rule", rule)
	}

	// Publish the updated Rule to the User's outbox
	if err := service.outboxService.Publish(session, model.FollowerTypeUser, rule.UserID, service.Activity(rule), model.NewAnonymousPermissions()); err != nil {
		return derp.Wrap(err, location, "Error publishing Rule", rule)
	}

	return nil
}
