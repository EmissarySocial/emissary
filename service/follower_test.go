package service

import (
	"context"
	"sort"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
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
	records         []model.Follower
	saved           []model.Follower // every record passed to Save, in order
	deleted         []model.Follower // every record passed to Delete, in order
	deleteError     error            // when set, Delete fails with this instead of deleting
	hardDeleteError error            // when set, HardDelete fails with this instead of deleting
}

// Context implements the data.Collection interface, returning a background context
func (c *followerCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *followerCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query fills the target with the IDs of every matching Follower, honoring a createDate sort.
// Only []model.IDOnly is supported, which is what QueryIDOnly -- and so every export -- asks for.
func (c *followerCollection) Query(target any, criteria exp.Expression, options ...option.Option) error {

	result, ok := target.(*[]model.IDOnly)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	// Collect every record the criteria admits
	matches := make([]model.Follower, 0)

	for _, record := range c.records {
		if matchesFollower(criteria, record) {
			matches = append(matches, record)
		}
	}

	// Apply the caller's sort order, then map down to IDs
	sortFollowers(matches, options...)

	for _, record := range matches {
		*result = append(*result, model.IDOnly{ID: record.FollowerID})
	}

	return nil
}

// sortFollowers orders records in place using a createDate SortOption, the only
// ordering that the Follower service asks this fake for
func sortFollowers(records []model.Follower, options ...option.Option) {

	for _, value := range options {

		sortOption, ok := value.(option.SortOption)

		if !ok {
			continue
		}

		if sortOption.FieldName != "createDate" {
			continue
		}

		sort.SliceStable(records, func(a int, b int) bool {

			if sortOption.IsDescending() {
				return records[a].CreateDate > records[b].CreateDate
			}

			return records[a].CreateDate < records[b].CreateDate
		})
	}
}

// Iterator returns every matching Follower, in insertion order
func (c *followerCollection) Iterator(criteria exp.Expression, _ ...option.Option) (data.Iterator, error) {

	result := make([]model.Follower, 0)

	for _, record := range c.records {
		if matchesFollower(criteria, record) {
			result = append(result, record)
		}
	}

	return &followerIterator{records: result}, nil
}

// followerIterator walks a fixed slice of Followers. Implements data.Iterator.
type followerIterator struct {
	records []model.Follower
	index   int
}

// Next copies the next Follower into the target, returning FALSE when the list is exhausted
func (i *followerIterator) Next(target any) bool {

	if i.index >= len(i.records) {
		return false
	}

	follower, ok := target.(*model.Follower)

	if !ok {
		return false
	}

	*follower = i.records[i.index]
	i.index++

	return true
}

// Count returns the number of records this iterator walks
func (i *followerIterator) Count() int { return len(i.records) }

// Close releases this iterator. It holds nothing.
func (i *followerIterator) Close() error { return nil }

// Error returns the error encountered while iterating, of which there are none
func (i *followerIterator) Error() error { return nil }

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

// Delete marks a Follower deleted, and remembers that it was asked to
func (c *followerCollection) Delete(object data.Object, _ string) error {

	if c.deleteError != nil {
		return c.deleteError
	}

	follower, ok := object.(*model.Follower)

	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	c.deleted = append(c.deleted, *follower)

	for index, record := range c.records {
		if record.FollowerID == follower.FollowerID {
			c.records[index].DeleteDate = 1
			return nil
		}
	}

	return nil
}

// HardDelete permanently drops every Follower that matches the criteria.  Unlike Delete,
// nothing is left behind to find, which is the whole point of the call under test.
func (c *followerCollection) HardDelete(criteria exp.Expression) error {

	if c.hardDeleteError != nil {
		return c.hardDeleteError
	}

	remaining := make([]model.Follower, 0, len(c.records))

	for _, record := range c.records {
		if !matchesFollower(criteria, record) {
			remaining = append(remaining, record)
		}
	}

	c.records = remaining
	return nil
}

// matchesFollower reports whether a Follower satisfies a criteria built from Equal and
// NotEqual predicates, which is every criteria the Follower service builds
func matchesFollower(criteria exp.Expression, record model.Follower) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		equals, supported := followerFieldEquals(predicate, record)

		// RULE: An unsupported field or value type is never a match, whichever
		// operator asked.  NotEqual must not turn "I don't know" into TRUE.
		if !supported {
			return false
		}

		switch predicate.Operator {

		case exp.OperatorEqual:
			return equals

		case exp.OperatorNotEqual:
			return !equals

		default:
			return false
		}
	})
}

// followerFieldEquals compares a Follower against a single predicate, returning whether the
// values are equal and whether the field and value type were understood at all
func followerFieldEquals(predicate exp.Predicate, record model.Follower) (bool, bool) {

	switch predicate.Field {

	case "_id":
		value, ok := predicate.Value.(primitive.ObjectID)
		return ok && record.FollowerID == value, ok

	case "parentId":
		value, ok := predicate.Value.(primitive.ObjectID)
		return ok && record.ParentID == value, ok

	case "type":
		value, ok := predicate.Value.(string)
		return ok && record.ParentType == value, ok

	case "method":
		value, ok := predicate.Value.(string)
		return ok && record.Method == value, ok

	case "stateId":
		value, ok := predicate.Value.(string)
		return ok && record.StateID == value, ok

	case "actor.emailAddress":
		value, ok := predicate.Value.(string)
		return ok && record.Actor.EmailAddress == value, ok

	case "deleteDate":
		value, ok := predicate.Value.(int)
		return ok && record.DeleteDate == int64(value), ok

	default:
		return false, false
	}
}

// followerSession hands out a single shared followerCollection
type followerSession struct {
	collection *followerCollection
	ctx        context.Context
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s followerSession) Collection(string) data.Collection { return s.collection }

// Context implements the data.Session interface, carrying the post-commit spool when the
// test supplied one
func (s followerSession) Context() context.Context {

	if s.ctx != nil {
		return s.ctx
	}

	return context.Background()
}

// Close implements the data.Session interface. The stub holds no resources to release.
func (s followerSession) Close() {}

// newFollowerService returns a Follower service backed by an in-memory set of Followers
func newFollowerService(followers ...model.Follower) (*Follower, followerSession) {

	service := NewFollower()

	return &service, followerSession{collection: &followerCollection{records: followers}}
}

// newSpooledFollowerService returns a Follower service whose session carries a post-commit
// spool, so that a test can read the tasks a call decided to enqueue
func newSpooledFollowerService(followers ...model.Follower) (*Follower, followerSession, *postcommit.Tasks) {

	service, session := newFollowerService(followers...)

	tasks := postcommit.NewTasks()
	session.ctx = postcommit.WithContext(context.Background(), tasks)

	return service, session, tasks
}

// taskNames returns the names of every task the spool holds, in order
func taskNames(tasks *postcommit.Tasks) []string {

	result := make([]string, 0)

	for _, task := range tasks.Drain() {
		result = append(result, task.Name)
	}

	return result
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

/******************************************
 * HardDeleteByID -- the undo path for an import
 ******************************************/

// newOwnedFollower returns a User Follower belonging to the provided parent
func newOwnedFollower(parentID primitive.ObjectID) model.Follower {

	follower := model.NewFollower()
	follower.ParentType = model.FollowerTypeUser
	follower.ParentID = parentID

	return follower
}

// TestFollower_HardDeleteByID_RemovesTheRecord is the second half of the same bug: the criteria
// queried `userId`, so UndoImport matched nothing and reported success anyway.
func TestFollower_HardDeleteByID_RemovesTheRecord(t *testing.T) {

	userID := primitive.NewObjectID()
	follower := newOwnedFollower(userID)

	service, session := newFollowerService(follower)

	require.Nil(t, service.HardDeleteByID(session, userID, follower.FollowerID))
	require.Empty(t, session.collection.records)
}

// TestFollower_HardDeleteByID_IsScopedToTheOwner confirms a valid FollowerID alone cannot
// reach across accounts
func TestFollower_HardDeleteByID_IsScopedToTheOwner(t *testing.T) {

	follower := newOwnedFollower(primitive.NewObjectID())

	service, session := newFollowerService(follower)

	require.Nil(t, service.HardDeleteByID(session, primitive.NewObjectID(), follower.FollowerID))
	require.Len(t, session.collection.records, 1)
}

// TestFollower_HardDeleteByID_LeavesSiblingsAlone proves the criteria removes one record and
// not every Follower the User owns
func TestFollower_HardDeleteByID_LeavesSiblingsAlone(t *testing.T) {

	userID := primitive.NewObjectID()
	doomed := newOwnedFollower(userID)
	survivor := newOwnedFollower(userID)

	service, session := newFollowerService(doomed, survivor)

	require.Nil(t, service.HardDeleteByID(session, userID, doomed.FollowerID))
	require.Len(t, session.collection.records, 1)
	require.Equal(t, survivor.FollowerID, session.collection.records[0].FollowerID)
}

// TestFollower_HardDeleteByID_UnknownIDIsNotAnError records that deleting a Follower that
// isn't there succeeds quietly, which is what makes the wrong-field bug so hard to see
func TestFollower_HardDeleteByID_UnknownIDIsNotAnError(t *testing.T) {

	userID := primitive.NewObjectID()
	follower := newOwnedFollower(userID)

	service, session := newFollowerService(follower)

	require.Nil(t, service.HardDeleteByID(session, userID, primitive.NewObjectID()))
	require.Len(t, session.collection.records, 1)
}

// TestFollower_HardDeleteByID_ReportsDatabaseFailure confirms a failed delete is wrapped and
// returned, rather than swallowed into the same silence as a query that matched nothing
func TestFollower_HardDeleteByID_ReportsDatabaseFailure(t *testing.T) {

	userID := primitive.NewObjectID()
	follower := newOwnedFollower(userID)

	service, session := newFollowerService(follower)
	session.collection.hardDeleteError = derp.Internal("test", "database is on fire")

	err := service.HardDeleteByID(session, userID, follower.FollowerID)

	require.NotNil(t, err)
	require.Len(t, session.collection.records, 1)
}
