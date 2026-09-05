package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests use a hand-built data.Collection fake instead of benpate/data-mock, which matches on
// the raw bson tag string and so cannot see `deleteDate` inside the inlined journal.Journal that
// every notDeleted() criteria queries.

/******************************************
 * In-Memory Fakes
 ******************************************/

// followerCollection is an in-memory data.Collection that holds model.Follower records
type followerCollection struct {
	records []model.Follower
	saved   []model.Follower // every record passed to Save, in order
}

// Context implements the data.Collection interface, returning a background context
func (c *followerCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *followerCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *followerCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *followerCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load copies the first matching Follower into the target
func (c *followerCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	for _, record := range c.records {

		if !matchesFollower(criteria, record) {
			continue
		}

		follower, ok := target.(*model.Follower)

		if !ok {
			return derp.Internal("test", "unexpected target type")
		}

		*follower = record
		return nil
	}

	return derp.NotFound("test", "not found")
}

// Save upserts a Follower, and remembers that it was asked to.  Tests that assert a code path
// leaves the database alone read `saved`, which is emptier than any error could be.
func (c *followerCollection) Save(object data.Object, _ string) error {

	follower, ok := object.(*model.Follower)

	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	c.saved = append(c.saved, *follower)

	for index, record := range c.records {
		if record.FollowerID == follower.FollowerID {
			c.records[index] = *follower
			return nil
		}
	}

	c.records = append(c.records, *follower)
	return nil
}

// Delete implements the data.Collection interface. Unused by these tests.
func (c *followerCollection) Delete(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *followerCollection) HardDelete(exp.Expression) error {
	return derp.Internal("test", "unused")
}

// matchesFollower reports whether a Follower satisfies a criteria on _id, parentId, method,
// actor.emailAddress, or deleteDate -- every field that LoadBySecret and LoadByEmailAddress query
func matchesFollower(criteria exp.Expression, record model.Follower) bool {

	// Any unsupported field or operator conservatively counts as "no match".
	return criteria.Match(func(predicate exp.Predicate) bool {

		if predicate.Operator != exp.OperatorEqual {
			return false
		}

		switch predicate.Field {

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.FollowerID == value

		case "parentId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.ParentID == value

		case "method":
			value, ok := predicate.Value.(string)
			return ok && record.Method == value

		case "actor.emailAddress":
			value, ok := predicate.Value.(string)
			return ok && record.Actor.EmailAddress == value

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && record.DeleteDate == int64(value)

		default:
			return false
		}
	})
}

// followerSession hands out a single shared followerCollection
type followerSession struct {
	collection *followerCollection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s followerSession) Collection(string) data.Collection { return s.collection }

// Context implements the data.Session interface, returning a background context
func (s followerSession) Context() context.Context { return context.Background() }

// Close implements the data.Session interface. The stub holds no resources to release.
func (s followerSession) Close() {}

// newFollowerService returns a Follower service backed by an in-memory set of Followers
func newFollowerService(followers ...model.Follower) (*Follower, followerSession) {

	service := NewFollower()

	return &service, followerSession{collection: &followerCollection{records: followers}}
}

// newSecretFollower returns a Follower of the provided method, carrying a known secret
func newSecretFollower(method string, secret string) model.Follower {

	follower := model.NewFollower()
	follower.Method = method
	follower.ParentType = model.FollowerTypeUser
	follower.Data.SetString("secret", secret)

	return follower
}

/******************************************
 * LoadBySecret
 ******************************************/

// TestFollowerLoadBySecret verifies the happy path: an email Follower who presents their own secret
func TestFollowerLoadBySecret(t *testing.T) {

	follower := newSecretFollower(model.FollowerMethodEmail, "abc123")
	followerService, session := newFollowerService(follower)

	result := model.NewFollower()
	require.NoError(t, followerService.LoadBySecret(session, follower.FollowerID, "abc123", &result))
	require.Equal(t, follower.FollowerID, result.FollowerID)
}

// TestFollowerLoadBySecret_RejectsOtherMethods verifies that only EMAIL Followers can be reached
// by secret.
//
// This is the authorization that Follower.UnsubscribeLink deliberately does NOT perform. The link
// is a public URL that anyone can type, so refusing to *render* it for an ActivityPub Follower
// would only have looked like a control. The query is the control: a non-email record does not
// load, even for a caller who somehow holds a matching secret.
func TestFollowerLoadBySecret_RejectsOtherMethods(t *testing.T) {

	follower := newSecretFollower(model.FollowerMethodActivityPub, "abc123")
	followerService, session := newFollowerService(follower)

	result := model.NewFollower()
	err := followerService.LoadBySecret(session, follower.FollowerID, "abc123", &result)

	require.Error(t, err)
	require.True(t, derp.IsNotFound(err))
	require.NotEqual(t, follower.FollowerID, result.FollowerID, "a rejected load must not populate the target")
}

// TestFollowerLoadBySecret_RejectsEmptySecret verifies that an absent secret is refused before the
// query runs.  Every Follower's secret would otherwise match "" if the field were ever unset.
func TestFollowerLoadBySecret_RejectsEmptySecret(t *testing.T) {

	follower := newSecretFollower(model.FollowerMethodEmail, "")
	followerService, session := newFollowerService(follower)

	result := model.NewFollower()
	err := followerService.LoadBySecret(session, follower.FollowerID, "", &result)

	require.Error(t, err)
	require.True(t, derp.IsForbidden(err))
}

// TestFollowerLoadBySecret_RejectsWrongSecret verifies that a valid FollowerID is not enough
func TestFollowerLoadBySecret_RejectsWrongSecret(t *testing.T) {

	follower := newSecretFollower(model.FollowerMethodEmail, "abc123")
	followerService, session := newFollowerService(follower)

	result := model.NewFollower()
	err := followerService.LoadBySecret(session, follower.FollowerID, "wrong-secret", &result)

	require.Error(t, err)
	require.True(t, derp.IsForbidden(err))
}

/******************************************
 * LoadByEmailAddress -- the inbound webhook's only lookup
 ******************************************/

// TestFollower_LoadByEmailAddress_IsScopedToTheParent guards the single check that bounds a
// leaked webhook secret to one account
func TestFollower_LoadByEmailAddress_IsScopedToTheParent(t *testing.T) {

	// The email address in a Mailchimp payload is attacker-supplied. An unscoped match would
	// let one forged (or misdirected) request remove any Follower on the server, for any
	// User -- so the parentId is part of the query, not a check applied afterward. D27.

	alice := primitive.NewObjectID()
	bob := primitive.NewObjectID()
	carol := primitive.NewObjectID()

	service, session := newFollowerService(
		newEmailFollower(alice, "shared@example.com"),
		newEmailFollower(bob, "shared@example.com"),
	)

	// Each owner reaches their OWN follower, and only that one
	for _, owner := range []primitive.ObjectID{alice, bob} {

		result := model.NewFollower()

		require.NoError(t, service.LoadByEmailAddress(session, owner, "shared@example.com", &result))
		require.Equal(t, owner, result.ParentID)
	}

	// A third party holding the same address reaches nobody
	result := model.NewFollower()
	err := service.LoadByEmailAddress(session, carol, "shared@example.com", &result)

	require.Error(t, err)
	require.True(t, derp.IsNotFound(err))
}

// TestFollower_LoadByEmailAddress_RefusesAnEmptyAddress guards the match that would return
// somebody else entirely
func TestFollower_LoadByEmailAddress_RefusesAnEmptyAddress(t *testing.T) {

	// A payload with no address would otherwise match the first Follower whose address was
	// never recorded -- every ActivityPub follower, for instance.

	alice := primitive.NewObjectID()
	service, session := newFollowerService(newEmailFollower(alice, ""))

	result := model.NewFollower()

	require.Error(t, service.LoadByEmailAddress(session, alice, "", &result))
}

// TestFollower_LoadByEmailAddress_OnlyReachesEmailFollowers confirms the method is part of
// the query
func TestFollower_LoadByEmailAddress_OnlyReachesEmailFollowers(t *testing.T) {

	// An ActivityPub actor can carry an email address in its profile. A Mailchimp unsubscribe
	// must never delete one: those two subscriptions are unrelated.

	alice := primitive.NewObjectID()

	activityPubFollower := newEmailFollower(alice, "person@example.com")
	activityPubFollower.Method = model.FollowerMethodActivityPub

	service, session := newFollowerService(activityPubFollower)

	result := model.NewFollower()
	err := service.LoadByEmailAddress(session, alice, "person@example.com", &result)

	require.Error(t, err)
	require.True(t, derp.IsNotFound(err))
}

// newEmailFollower returns an EMAIL Follower of the provided parent, carrying an address
func newEmailFollower(parentID primitive.ObjectID, emailAddress string) model.Follower {

	follower := model.NewFollower()
	follower.ParentID = parentID
	follower.Method = model.FollowerMethodEmail
	follower.Actor.EmailAddress = emailAddress

	return follower
}
