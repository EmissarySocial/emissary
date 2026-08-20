package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PushSubscription manages Web Push subscription records (one per browser registration per User).
type PushSubscription struct{}

// NewPushSubscription returns a fully initialized PushSubscription service
func NewPushSubscription() PushSubscription {
	return PushSubscription{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *PushSubscription) Refresh(factory *Factory) {
	// Nothing to refresh.
}

// Close stops any background processes controlled by this service
func (service *PushSubscription) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the PushSubscription collection for the provided database session
func (service *PushSubscription) collection(session data.Session) data.Collection {
	return session.Collection("PushSubscription")
}

// Count returns the number of PushSubscriptions that match the provided criteria
func (service *PushSubscription) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice of PushSubscriptions that match the provided criteria
func (service *PushSubscription) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.PushSubscription, error) {
	result := make([]model.PushSubscription, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator of PushSubscriptions that match the provided criteria
func (service *PushSubscription) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc over the PushSubscriptions that match the provided criteria
func (service *PushSubscription) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.PushSubscription], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.PushSubscription.Range", "Creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewPushSubscription), nil
}

// Load retrieves a PushSubscription from the database
func (service *PushSubscription) Load(session data.Session, criteria exp.Expression, sub *model.PushSubscription) error {

	if err := service.collection(session).Load(notDeleted(criteria), sub); err != nil {
		return derp.Wrap(err, "service.PushSubscription.Load", "Loading PushSubscription", criteria)
	}

	return nil
}

// Save adds/updates a PushSubscription in the database
func (service *PushSubscription) Save(session data.Session, sub *model.PushSubscription, note string) error {

	const location = "service.PushSubscription.Save"

	if _, err := service.Schema().Validate(sub); err != nil {
		return derp.Wrap(err, location, "Validating PushSubscription", sub)
	}

	if err := service.collection(session).Save(sub, note); err != nil {
		return derp.Wrap(err, location, "Saving PushSubscription", sub, note)
	}

	return nil
}

// Delete removes a PushSubscription from the database (hard delete)
func (service *PushSubscription) Delete(session data.Session, sub *model.PushSubscription, note string) error {

	const location = "service.PushSubscription.Delete"

	// Hard delete, never virtual: every value here is minted by the browser, so a tombstone could
	// never be resurrected -- it would only retain a device identifier and its secrets after the
	// User asked us to stop pushing to them.
	if err := service.collection(session).HardDelete(exp.Equal("_id", sub.PushSubscriptionID)); err != nil {
		return derp.Wrap(err, location, "Deleting PushSubscription", sub, note)
	}

	// This subscription has left the building.
	return nil
}

// Schema returns the rosetta schema that describes a PushSubscription
func (service *PushSubscription) Schema() schema.Schema {
	return schema.New(model.PushSubscriptionSchema())
}

/******************************************
 * Custom Queries + Behaviors
 ******************************************/

// RangeByUserID iterates over every PushSubscription owned by the provided User.
func (service *PushSubscription) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.PushSubscription], error) {
	return service.Range(session, exp.Equal("userId", userID))
}

// LoadByEndpoint loads a PushSubscription by its (unique) endpoint URL.
func (service *PushSubscription) LoadByEndpoint(session data.Session, endpoint string, sub *model.PushSubscription) error {
	return service.Load(session, exp.Equal("endpoint", endpoint), sub)
}

// Upsert creates or updates a PushSubscription for a User, keyed by endpoint
func (service *PushSubscription) Upsert(session data.Session, userID primitive.ObjectID, endpoint string, p256dh string, auth string, userAgent string) error {

	const location = "service.PushSubscription.Upsert"

	// Load any subscription that already claims this endpoint
	sub := model.NewPushSubscription()

	if err := service.LoadByEndpoint(session, endpoint, &sub); err != nil {

		// A NotFound just means the endpoint is unclaimed, which is the normal path for a new
		// registration; anything else is a real database failure.
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Loading PushSubscription", endpoint)
		}
	}

	// RULE: An endpoint identifies a BROWSER, so only its current owner may re-bind it.  The row is
	// globally unique, so rebinding would transfer it, silently ending the previous owner's push.
	if !sub.IsNew() && (sub.UserID != userID) {
		return errEndpointConflict(location, endpoint)
	}

	// Bind the subscription to this User, refreshing the keys that the browser minted
	sub.UserID = userID
	sub.Endpoint = endpoint
	sub.P256DH = p256dh
	sub.Auth = auth
	sub.UserAgent = userAgent

	// Save the subscription
	if err := service.Save(session, &sub, "Upsert push subscription"); err != nil {

		// A lost creation race trips the unique endpoint index: a concurrent request inserted this
		// endpoint between our Load and our Save.  Re-apply the ownership rule against the winner
		// rather than retrying blindly, which would hand the loser the winner's subscription.
		if derp.IsConflict(err) {
			return service.upsertOntoRaceWinner(session, &sub, userID, endpoint)
		}

		return derp.Wrap(err, location, "Saving PushSubscription", endpoint)
	}

	return nil
}

// upsertOntoRaceWinner completes an Upsert that lost a creation race, folding onto the winner's record
func (service *PushSubscription) upsertOntoRaceWinner(session data.Session, sub *model.PushSubscription, userID primitive.ObjectID, endpoint string) error {

	const location = "service.PushSubscription.upsertOntoRaceWinner"

	// Load the record that won the race
	existing := model.NewPushSubscription()

	if err := service.LoadByEndpoint(session, endpoint, &existing); err != nil {
		return derp.Wrap(err, location, "Re-loading PushSubscription after duplicate-key conflict", endpoint)
	}

	// RULE: the winner may belong to a different User, so the ownership rule applies here too
	if existing.UserID != userID {
		return errEndpointConflict(location, endpoint)
	}

	// Adopt the winner's identity, which makes the Save below update in place instead of inserting
	// a second row.  Reaching here means the same User registered twice at once.
	sub.PushSubscriptionID = existing.PushSubscriptionID
	sub.Journal = existing.Journal

	if err := service.Save(session, sub, "Upsert push subscription"); err != nil {
		return derp.Wrap(err, location, "Saving PushSubscription after duplicate-key conflict", endpoint)
	}

	// Second place is still a winner.
	return nil
}

// errEndpointConflict returns the (409) Conflict for an endpoint that belongs to a different User
func errEndpointConflict(location string, endpoint string) error {

	// RULE: Conflict means "someone else owns this endpoint", and the browser answers it by retiring
	// its subscription and registering a fresh one.  Never widen this to the Forbidden that
	// PostPushSubscription returns for a disallowed endpoint, which the browser must not retry.
	return derp.Conflict(location, "This push endpoint is already registered to a different User", endpoint)
}

// DeleteByEndpoint removes the PushSubscription with the provided endpoint (e.g. after a 404/410
// from the push service, or on explicit unsubscribe)
func (service *PushSubscription) DeleteByEndpoint(session data.Session, endpoint string, note string) error {

	const location = "service.PushSubscription.DeleteByEndpoint"

	// HardDelete matches nothing when the endpoint is already gone, which makes this idempotent.
	if err := service.collection(session).HardDelete(exp.Equal("endpoint", endpoint)); err != nil {
		return derp.Wrap(err, location, "Deleting PushSubscription", endpoint, note)
	}

	return nil
}

// DeleteByUserID removes every PushSubscription owned by the provided User (account teardown).
func (service *PushSubscription) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.PushSubscription.DeleteByUserID"

	if err := service.collection(session).HardDelete(exp.Equal("userId", userID)); err != nil {
		return derp.Wrap(err, location, "Deleting PushSubscriptions", userID, note)
	}

	return nil
}
