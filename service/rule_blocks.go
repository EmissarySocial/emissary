package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/benpate/data"
)

// BlockActor creates a new Rule record to block the specified actor for the user.
// If such a record already exists, then no action is taken.
func (service *Rule) BlockActor(session data.Session, userID primitive.ObjectID, actorID string, note string) error {

	const location = "service.Rule.BlockActor"

	// Try to load the existing Rule record for this user and actor
	rule := model.NewRule()
	err := service.loadActorRule(session, userID, actorID, &rule)

	// If the record already exists, then there you go.
	if err == nil {
		return nil
	}

	// Report legitimate errors
	if !derp.IsNotFound(err) {
		return derp.Wrap(err, location, "Loading rule for user and actor", userID, actorID)
	}

	// Otherwise, create a new Rule record to block this actor
	rule.UserID = userID
	rule.Type = model.RuleTypeActor
	rule.Trigger = actorID
	rule.Action = model.RuleActionBlock
	rule.Note = note

	if err := service.Save(session, &rule, note); err != nil {
		return derp.Wrap(err, location, "Saving rule for user and URL", rule)
	}

	// Success.
	return nil
}

// UnblockActor removes any existing Rule record that is blocking the specified actor for the user.
// If no such record exists, then no action is taken.
func (service *Rule) UnblockActor(session data.Session, userID primitive.ObjectID, actorID string) error {

	const location = "service.Rule.UnblockActor"

	// Try to load the existing Rule record for this user and actor
	rule := model.NewRule()

	if err := service.loadActorRule(session, userID, actorID, &rule); err != nil {

		// If the record is not found, then there is nothing to unblock.
		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Loading rule for user and actor", userID, actorID)
	}

	// Delete the existing Rule
	if err := service.Delete(session, &rule, "Unblocking actor"); err != nil {
		return derp.Wrap(err, location, "Deleting rule for user and actor", rule)
	}

	// Success
	return nil
}

// loadActorRule finds the User's ACTOR Rule for the provided address, whatever friendly form the
// caller holds (a webfinger @user@host handle, an alias URL, or the canonical actor URL) and
// whichever key shape the stored Rule carries. Returns NotFound when no Rule matches.
//
// TWO key shapes exist on disk, which is why this cannot be a single lookup:
//
//   - Rules saved through Rule.Save key on the CANONICAL actor URL (Save resolves the Trigger first).
//   - Rules backfilled by the v027 migration key on the RAW Trigger the User typed -- that migration
//     runs offline over the whole collection and cannot resolve handles over the network.
//
// Probing only the raw address is the bug this exists to prevent: BlockActor stores a handle-derived
// Trigger that Save canonicalizes, so a later UnblockActor keyed on the same handle found nothing and
// returned "nothing to unblock" -- an unblock that never unblocked, with no error to show for it.
func (service *Rule) loadActorRule(session data.Session, userID primitive.ObjectID, actorID string, rule *model.Rule) error {

	const location = "service.Rule.loadActorRule"

	// Probe the canonical actor URL first -- the shape every Rule saved through the service carries.
	//
	// RULE: a resolution failure is NOT fatal here. An actor whose server has gone offline (or been
	// defederated) must still be unblockable, so an unresolvable address falls through to the raw
	// probe below rather than stranding the User's own Rule behind a remote fetch they cannot fix.
	if canonical, err := service.resolveActorAddress(actorID); err == nil {

		err := service.LoadByMatchKey(session, userID, model.RuleTypeActor, canonical, rule)

		if err == nil {
			return nil
		}

		// A miss under the canonical key only means the Rule predates trigger resolution; any other
		// failure is a real database problem that the fallback probe must not mask.
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Loading rule by canonical actor URL", userID, canonical)
		}
	}

	// Fall back to the address exactly as supplied, which is the shape a legacy (v027-backfilled)
	// Rule carries -- and the only shape available when the address will not resolve.
	return service.LoadByMatchKey(session, userID, model.RuleTypeActor, actorID, rule)
}
