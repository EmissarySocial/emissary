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
		return nil, derp.Wrap(err, "service.PushSubscription.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewPushSubscription), nil
}

// Load retrieves a PushSubscription from the database
func (service *PushSubscription) Load(session data.Session, criteria exp.Expression, sub *model.PushSubscription) error {

	if err := service.collection(session).Load(notDeleted(criteria), sub); err != nil {
		return derp.Wrap(err, "service.PushSubscription.Load", "Unable to load PushSubscription", criteria)
	}

	return nil
}

// Save adds/updates a PushSubscription in the database
func (service *PushSubscription) Save(session data.Session, sub *model.PushSubscription, note string) error {

	const location = "service.PushSubscription.Save"

	if _, err := service.Schema().Validate(sub); err != nil {
		return derp.Wrap(err, location, "Unable to validate PushSubscription", sub)
	}

	if err := service.collection(session).Save(sub, note); err != nil {
		return derp.Wrap(err, location, "Unable to save PushSubscription", sub, note)
	}

	return nil
}

// Delete removes a PushSubscription from the database (hard delete)
//
// RULE: PushSubscriptions are hard-deleted, never virtual-deleted.  Every value in the record is
// minted by the browser (the endpoint and its crypto keys), so a tombstone could never be
// resurrected -- re-enabling push always mints a fresh subscription.  It would only retain a device
// identifier and its secrets after the User asked us to stop pushing to them.
func (service *PushSubscription) Delete(session data.Session, sub *model.PushSubscription, note string) error {

	const location = "service.PushSubscription.Delete"

	if err := service.collection(session).HardDelete(exp.Equal("_id", sub.PushSubscriptionID)); err != nil {
		return derp.Wrap(err, location, "Unable to delete PushSubscription", sub, note)
	}

	return nil
}

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

// Upsert creates or updates a PushSubscription for a User, keyed by endpoint.  The userID is taken
// from the authenticated session (never the request body), so one User cannot overwrite another's
// subscription.
func (service *PushSubscription) Upsert(session data.Session, userID primitive.ObjectID, endpoint string, p256dh string, auth string, userAgent string) error {

	const location = "service.PushSubscription.Upsert"

	sub := model.NewPushSubscription()
	err := service.LoadByEndpoint(session, endpoint, &sub)

	if err != nil && !derp.IsNotFound(err) {
		return derp.Wrap(err, location, "Unable to load PushSubscription", endpoint)
	}

	// Whether new or existing, (re)bind it to THIS user and refresh the keys.
	sub.UserID = userID
	sub.Endpoint = endpoint
	sub.P256DH = p256dh
	sub.Auth = auth
	sub.UserAgent = userAgent

	if err := service.Save(session, &sub, "Upsert push subscription"); err != nil {
		return derp.Wrap(err, location, "Unable to save PushSubscription", endpoint)
	}

	return nil
}

// DeleteByEndpoint removes the PushSubscription with the provided endpoint (e.g. after a 404/410
// from the push service, or on explicit unsubscribe).  Deleting an endpoint that is already gone
// is not an error.
func (service *PushSubscription) DeleteByEndpoint(session data.Session, endpoint string, note string) error {

	const location = "service.PushSubscription.DeleteByEndpoint"

	if err := service.collection(session).HardDelete(exp.Equal("endpoint", endpoint)); err != nil {
		return derp.Wrap(err, location, "Unable to delete PushSubscription", endpoint, note)
	}

	return nil
}

// DeleteByUserID removes every PushSubscription owned by the provided User (account teardown).
func (service *PushSubscription) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.PushSubscription.DeleteByUserID"

	if err := service.collection(session).HardDelete(exp.Equal("userId", userID)); err != nil {
		return derp.Wrap(err, location, "Unable to delete PushSubscriptions", userID, note)
	}

	return nil
}
