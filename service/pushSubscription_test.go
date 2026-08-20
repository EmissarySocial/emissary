package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * subStore — an in-memory data.Collection that matches PushSubscriptions on the
 * fields Upsert queries: endpoint, userId, deleteDate.
 *
 * Hand-built for the same reason as collectionItem_test.go: benpate/data-mock matches on the raw
 * bson tag string, and PushSubscription.UserAgent carries `,omitempty`.
 *
 * NOTE: a fake cannot enforce a real unique index, so these tests pin the SERVICE's decisions --
 * the ownership rule, and its reaction to a duplicate-key error -- not the index itself.  The index
 * lives in queries/sync/pushSubscription.go and needs a live database to exercise.
 ******************************************/

// subStore is an in-memory data.Collection of PushSubscriptions, used by the tests in this file
type subStore struct {
	records []*model.PushSubscription

	// duplicateKeyOnce simulates losing a creation race: the next insert fails with a duplicate-key
	// error, exactly as the unique endpoint index would report it.
	duplicateKeyOnce bool
}

// Context implements the interface, returning a background context
func (c *subStore) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *subStore) Count(criteria exp.Expression, _ ...option.Option) (int64, error) {
	var count int64
	for _, record := range c.records {
		if matchesSub(criteria, record) {
			count = count + 1
		}
	}
	return count, nil
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *subStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *subStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

// Load implements the data.Collection interface, backed by this stub's in-memory records
func (c *subStore) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	for _, record := range c.records {
		if matchesSub(criteria, record) {
			sub, ok := target.(*model.PushSubscription)
			if !ok {
				return derp.Internal("test", "unexpected target type")
			}
			*sub = *record
			return nil
		}
	}

	return derp.NotFound("test", "not found")
}

// Save implements the data.Collection interface, backed by this stub's in-memory records
func (c *subStore) Save(object data.Object, _ string) error {

	sub, ok := object.(*model.PushSubscription)

	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	// Update in place when this record already exists.
	for index, record := range c.records {
		if record.PushSubscriptionID == sub.PushSubscriptionID {
			saved := *sub
			c.records[index] = &saved
			return nil
		}
	}

	// Otherwise INSERT. The unique endpoint index rejects a second live row for the same endpoint.
	if c.duplicateKeyOnce {
		c.duplicateKeyOnce = false
		return duplicateKeyError()
	}

	saved := *sub
	saved.CreateDate = 1 // mark as persisted (IsNew keys off CreateDate)
	c.records = append(c.records, &saved)
	return nil
}

// Delete implements the data.Collection interface. Unused by these tests.
func (c *subStore) Delete(data.Object, string) error { return nil }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *subStore) HardDelete(criteria exp.Expression) error {
	remaining := make([]*model.PushSubscription, 0, len(c.records))
	for _, record := range c.records {
		if !matchesSub(criteria, record) {
			remaining = append(remaining, record)
		}
	}
	c.records = remaining
	return nil
}

// matchesSub reports whether a record satisfies an equality criteria on endpoint/userId/_id.
func matchesSub(criteria exp.Expression, record *model.PushSubscription) bool {

	// Any unsupported field or operator conservatively counts as "no match".
	return criteria.Match(func(predicate exp.Predicate) bool {

		if predicate.Operator != exp.OperatorEqual {
			return false
		}

		switch predicate.Field {

		case "endpoint":
			value, ok := predicate.Value.(string)
			return ok && record.Endpoint == value

		case "userId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.UserID == value

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.PushSubscriptionID == value

		case "deleteDate":
			// Nothing is ever soft-deleted (PushSubscriptions are hard-deleted), so the
			// notDeleted() guard that every read carries always holds.
			return true
		}

		return false
	})
}

// subSession is a data.Session that hands out a single subStore
type subSession struct {
	collection *subStore
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s subSession) Collection(string) data.Collection { return s.collection }

// Context implements the interface, returning a background context
func (s subSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s subSession) Close() {}

// newSubService returns a PushSubscription service backed by the provided store
func newSubService(store *subStore) (*PushSubscription, subSession) {
	service := NewPushSubscription()
	return &service, subSession{collection: store}
}

// seed puts a live subscription for `userID` at `endpoint` directly into the store.
func (c *subStore) seed(userID primitive.ObjectID, endpoint string) *model.PushSubscription {
	sub := model.NewPushSubscription()
	sub.UserID = userID
	sub.Endpoint = endpoint
	sub.P256DH = "seed-p256dh"
	sub.Auth = "seed-auth"
	sub.CreateDate = 1
	c.records = append(c.records, &sub)
	return &sub
}

/******************************************
 * Upsert: the ownership rule
 ******************************************/

// TestPushSubscription_Upsert_RefusesAnotherUsersEndpoint pins the reported vulnerability
//
// The endpoint is globally unique, so an unchecked upsert TRANSFERS the single row and the victim
// silently loses push delivery.
func TestPushSubscription_Upsert_RefusesAnotherUsersEndpoint(t *testing.T) {

	store := &subStore{}
	service, session := newSubService(store)

	victim := primitive.NewObjectID()
	attacker := primitive.NewObjectID()
	const endpoint = "https://push.example.test/victim-endpoint"

	store.seed(victim, endpoint)

	err := service.Upsert(session, attacker, endpoint, "attacker-p256dh", "attacker-auth", "attacker-agent")

	require.NotNil(t, err, "claiming another User's endpoint must be refused")
	require.Equal(t, http.StatusConflict, derp.ErrorCode(err), "must be Conflict: the browser re-registers on Conflict, but must leave a Forbidden endpoint alone")

	// The victim's row must be untouched: same owner, same keys.
	require.Len(t, store.records, 1)
	require.Equal(t, victim, store.records[0].UserID, "the victim must still own this subscription")
	require.Equal(t, "seed-p256dh", store.records[0].P256DH, "the attacker's keys must not have been written")

	// The handler wraps this error and errorHandler uses derp.ErrorCode as the literal HTTP status,
	// so a 409 lost in the wrap would show the browser a 500 and never fire its recovery path.
	wrapped := derp.Wrap(err, "handler.PostPushSubscription", "Saving push subscription")
	require.Equal(t, http.StatusConflict, derp.ErrorCode(wrapped), "the 409 must survive derp.Wrap")
}

// TestPushSubscription_Upsert_InsertsNewEndpoint confirms the guard does not block the normal case
func TestPushSubscription_Upsert_InsertsNewEndpoint(t *testing.T) {

	store := &subStore{}
	service, session := newSubService(store)

	userID := primitive.NewObjectID()

	require.Nil(t, service.Upsert(session, userID, "https://push.example.test/new", "p", "a", "agent"))

	require.Len(t, store.records, 1)
	require.Equal(t, userID, store.records[0].UserID)
}

// TestPushSubscription_Upsert_SameUserRefreshesKeys confirms a User re-registering their OWN
// endpoint still updates in place
//
// This is the ordinary path -- a browser re-subscribing rotates its keys -- so the ownership guard
// must not catch it.
func TestPushSubscription_Upsert_SameUserRefreshesKeys(t *testing.T) {

	store := &subStore{}
	service, session := newSubService(store)

	userID := primitive.NewObjectID()
	const endpoint = "https://push.example.test/mine"

	seeded := store.seed(userID, endpoint)

	require.Nil(t, service.Upsert(session, userID, endpoint, "fresh-p256dh", "fresh-auth", "fresh-agent"))

	require.Len(t, store.records, 1, "must update in place, not insert a second row")
	require.Equal(t, seeded.PushSubscriptionID, store.records[0].PushSubscriptionID)
	require.Equal(t, "fresh-p256dh", store.records[0].P256DH)
	require.Equal(t, "fresh-auth", store.records[0].Auth)
}

/******************************************
 * Upsert: losing a creation race
 ******************************************/

// TestPushSubscription_Upsert_RaceWithSameUserFolds confirms the SAME User losing a creation race
// folds onto the winner's record instead of erroring
func TestPushSubscription_Upsert_RaceWithSameUserFolds(t *testing.T) {

	store := &subStore{}
	service, session := newSubService(store)

	userID := primitive.NewObjectID()
	const endpoint = "https://push.example.test/raced"

	// The winner's row lands between our Load and our Save, so the insert hits the unique index.
	winner := store.seed(userID, endpoint)
	store.duplicateKeyOnce = true

	require.Nil(t, service.Upsert(session, userID, endpoint, "fresh-p256dh", "fresh-auth", "agent"))

	require.Len(t, store.records, 1)
	require.Equal(t, winner.PushSubscriptionID, store.records[0].PushSubscriptionID)
	require.Equal(t, "fresh-p256dh", store.records[0].P256DH)
}

// TestPushSubscription_Upsert_RaceWithOtherUserConflicts is the security half of the race
//
// Losing to a DIFFERENT User must re-apply the ownership rule.  A blind retry would hand the loser
// the winner's subscription -- the very transfer the guard exists to stop.
func TestPushSubscription_Upsert_RaceWithOtherUserConflicts(t *testing.T) {

	store := &subStore{}
	service, session := newSubService(store)

	victim := primitive.NewObjectID()
	attacker := primitive.NewObjectID()
	const endpoint = "https://push.example.test/raced"

	// Nothing exists when the attacker's Upsert Loads; the victim's row lands before its Save.
	store.duplicateKeyOnce = true
	seeded := store.seed(victim, endpoint)

	err := service.Upsert(session, attacker, endpoint, "attacker-p256dh", "attacker-auth", "agent")

	require.NotNil(t, err)
	require.Equal(t, http.StatusConflict, derp.ErrorCode(err))

	require.Len(t, store.records, 1)
	require.Equal(t, victim, store.records[0].UserID, "the race winner must keep their subscription")
	require.Equal(t, seeded.PushSubscriptionID, store.records[0].PushSubscriptionID)
	require.Equal(t, "seed-p256dh", store.records[0].P256DH)
}
