package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RuleSuppression defines a service that manages the don't-re-import records for deleted
// imported Rules (P7-3). See model.RuleSuppression for the why.
type RuleSuppression struct{}

// NewRuleSuppression returns a fully initialized RuleSuppression service
func NewRuleSuppression() RuleSuppression {
	return RuleSuppression{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *RuleSuppression) Refresh(factory *Factory) {
	// No dependencies. Simplicity is its own reward.
}

// Close stops any background processes controlled by this service
func (service *RuleSuppression) Close() {
	// Nothing to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the RuleSuppression collection for the provided database session
func (service *RuleSuppression) collection(session data.Session) data.Collection {
	return session.Collection("RuleSuppression")
}

// Save adds/updates a RuleSuppression in the database
func (service *RuleSuppression) Save(session data.Session, suppression *model.RuleSuppression, note string) error {

	if err := service.collection(session).Save(suppression, note); err != nil {
		return derp.Wrap(err, "service.RuleSuppression.Save", "Saving RuleSuppression", suppression)
	}

	return nil
}

// Load retrieves a RuleSuppression from the database
func (service *RuleSuppression) Load(session data.Session, criteria exp.Expression, suppression *model.RuleSuppression) error {

	if err := service.collection(session).Load(notDeleted(criteria), suppression); err != nil {
		return derp.Wrap(err, "service.RuleSuppression.Load", "Loading RuleSuppression", criteria)
	}

	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

// Suppress records that the provided remote moderation entry must never re-import for this
// owner (P7-3). Idempotent: suppressing an already-suppressed entry is a no-op, so the unique
// {userId, remoteId} index never trips on a double delete.
func (service *RuleSuppression) Suppress(session data.Session, userID primitive.ObjectID, followingID primitive.ObjectID, remoteID string) error {

	const location = "service.RuleSuppression.Suppress"

	// RULE: an entry with no remote identity has nothing to suppress
	if remoteID == "" {
		return nil
	}

	// An existing suppression already does the job
	suppressed, err := service.IsSuppressed(session, userID, remoteID)

	if err != nil {
		return derp.Wrap(err, location, "Checking existing suppression", remoteID)
	}

	if suppressed {
		return nil
	}

	// Write the don't-re-import record
	suppression := model.NewRuleSuppression()
	suppression.UserID = userID
	suppression.FollowingID = followingID
	suppression.RemoteID = remoteID

	if err := service.Save(session, &suppression, "Suppressing re-import of deleted Rule"); err != nil {
		return derp.Wrap(err, location, "Saving RuleSuppression", remoteID)
	}

	// Gone, and it's STAYING gone.
	return nil
}

// IsSuppressed returns TRUE if the provided remote moderation entry has been suppressed for this
// owner. The subscription backfill (Phase 7C) consults this before importing every entry.
func (service *RuleSuppression) IsSuppressed(session data.Session, userID primitive.ObjectID, remoteID string) (bool, error) {

	const location = "service.RuleSuppression.IsSuppressed"

	criteria := exp.Equal("userId", userID).AndEqual("remoteId", remoteID)
	suppression := model.NewRuleSuppression()

	if err := service.Load(session, criteria, &suppression); err != nil {

		// Not found simply means "not suppressed"
		if derp.IsNotFound(err) {
			return false, nil
		}

		// Any other error is a genuine failure to reach the database
		return false, derp.Wrap(err, location, "Loading RuleSuppression", remoteID)
	}

	return true, nil
}

// DeleteByFollowingID hard-deletes every suppression tied to the provided subscription. Called
// when a provider subscription is removed outright -- with no subscription there is no backfill
// to suppress, and a future re-subscribe starts from a clean slate.
func (service *RuleSuppression) DeleteByFollowingID(session data.Session, followingID primitive.ObjectID) error {

	const location = "service.RuleSuppression.DeleteByFollowingID"

	criteria := exp.Equal("followingId", followingID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting RuleSuppressions", followingID)
	}

	return nil
}
