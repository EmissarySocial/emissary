package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/benpate/data"
)

// BlockActor creates a new Rule record to block the specified actor for the user.
// If such a record already exists, then no action is taken.
func (service *Rule) BlockActor(session data.Session, userID primitive.ObjectID, actorID string, note string) error {

	const location = "service.Rule.BlockActor"
	spew.Dump(location, "Attempting to block actor", actorID)

	// Try to load the existing Rule record for this user and URL
	rule := model.NewRule()
	err := service.LoadByTrigger(session, userID, model.RuleTypeActor, actorID, &rule)

	// If the record already exists, then there you go.
	if err == nil {
		return nil
	}

	// Report legitimate errors
	if !derp.IsNotFound(err) {
		return derp.Wrap(err, location, "Unable to load rule for user and URL", userID, actorID)
	}

	// Otherwise, create a new Rule record to block this actor
	rule.UserID = userID
	rule.Type = model.RuleTypeActor
	rule.Trigger = actorID
	rule.Action = model.RuleActionBlock
	rule.Note = note

	if err := service.Save(session, &rule, note); err != nil {
		return derp.Wrap(err, location, "Unable to save rule for user and URL", rule)
	}

	// Success.
	return nil
}

// UnblockActor removes any existing Rule record that is blocking the specified actor for the user.
// If no such record exists, then no action is taken.
func (service *Rule) UnblockActor(session data.Session, userID primitive.ObjectID, actorID string) error {

	const location = "service.Rule.UnblockActor"
	spew.Dump(location, "Attempting to unblock actor", actorID)

	// Try to load the existing Rule record for this user and URL
	rule := model.NewRule()
	err := service.LoadByTrigger(session, userID, model.RuleTypeActor, actorID, &rule)

	if err != nil {

		// If the record is not found, then there is nothing to unblock.
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Unable to load rule for user and URL", userID, actorID)
	}

	// Delete the existing Rule
	if err := service.Delete(session, &rule, "Unblocking actor"); err != nil {
		return derp.Wrap(err, location, "Unable to delete rule for user and URL", rule)
	}

	// Success
	return nil
}
