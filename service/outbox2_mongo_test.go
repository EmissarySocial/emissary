package service

import (
	"context"
	"slices"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	querysync "github.com/EmissarySocial/emissary/queries/sync"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests run the Outbox2 queries against a real MongoDB with the production indexes, because
// what they guard is the filter each query sends. They skip when no database is reachable.

// outbox2QueryTest holds an Outbox2 service and the session its rows are written through
type outbox2QueryTest struct {
	service *Outbox2
	session data.Session
}

// newOutbox2QueryTest returns an Outbox2 service over a throwaway database that carries the Outbox2 indexes
func newOutbox2QueryTest(t *testing.T) outbox2QueryTest {

	t.Helper()

	database := newDomainTestDatabase(t)
	require.NoError(t, querysync.Outbox2(context.Background(), database))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	session, err := mongodb.NewServer(database).Session(ctx)
	require.NoError(t, err)

	return outbox2QueryTest{service: &Outbox2{}, session: session}
}

// save writes one OutboxItem directly, with a fixed createDate so that paging is deterministic
func (test outbox2QueryTest) save(t *testing.T, actorType string, actorID primitive.ObjectID, createDate int64, permissions model.Permissions, mutate func(*model.OutboxItem)) model.OutboxItem {

	t.Helper()

	item := model.NewOutboxItem()
	item.ActorType = actorType
	item.ActorID = actorID
	item.URL = "https://example.test/@" + actorID.Hex() + "/pub/outbox/" + item.ActivityID.Hex()
	item.Permissions = permissions
	item.CreateDate = createDate

	if mutate != nil {
		mutate(&item)
	}

	require.NoError(t, test.session.Collection("Outbox2").Save(&item, "test"))
	return item
}

// activityIDs returns the IDs of the provided items, in order
func activityIDs(items []model.OutboxItem) []primitive.ObjectID {

	result := make([]primitive.ObjectID, 0, len(items))

	for _, item := range items {
		result = append(result, item.ActivityID)
	}

	return result
}

// TestOutbox2_QueryByParentAndDate verifies paging, ordering, actor scoping and the permission filter
func TestOutbox2_QueryByParentAndDate(t *testing.T) {

	test := newOutbox2QueryTest(t)

	userID := primitive.NewObjectID()
	groupID := primitive.NewObjectID()
	anonymous := model.NewAnonymousPermissions()

	public1 := test.save(t, model.ActorTypeUser, userID, 100, anonymous, nil)
	public2 := test.save(t, model.ActorTypeUser, userID, 200, anonymous, nil)
	public3 := test.save(t, model.ActorTypeUser, userID, 300, anonymous, nil)
	members := test.save(t, model.ActorTypeUser, userID, 250, model.Permissions{groupID}, nil)
	unreadable := test.save(t, model.ActorTypeUser, userID, 260, model.NewPermissions(), nil)

	// Rows that must never appear: another actor, the same ID as a Stream, and a soft-deleted row
	test.save(t, model.ActorTypeUser, primitive.NewObjectID(), 150, anonymous, nil)
	test.save(t, model.ActorTypeStream, userID, 150, anonymous, nil)
	test.save(t, model.ActorTypeUser, userID, 150, anonymous, func(item *model.OutboxItem) { item.DeleteDate = 999 })

	query := func(permissions model.Permissions, maxDate int64, maxRows int) []primitive.ObjectID {
		result, err := test.service.QueryByParentAndDate(test.session, model.ActorTypeUser, userID, permissions, maxDate, maxRows)
		require.NoError(t, err)
		return activityIDs(result)
	}

	// An anonymous caller sees only public items, newest first
	require.Equal(t, []primitive.ObjectID{public3.ActivityID, public2.ActivityID, public1.ActivityID}, query(anonymous, 1000, 60))

	// A group member also sees the group's item, in date order
	require.Equal(t,
		[]primitive.ObjectID{public3.ActivityID, members.ActivityID, public2.ActivityID, public1.ActivityID},
		query(append(model.NewAnonymousPermissions(), groupID), 1000, 60))

	// maxDate is exclusive, and maxRows caps the page
	require.Equal(t, []primitive.ObjectID{public2.ActivityID, public1.ActivityID}, query(anonymous, 300, 60))
	require.Equal(t, []primitive.ObjectID{public3.ActivityID}, query(anonymous, 1000, 1))

	// RULE: a caller with no permissions sees nothing, and an item with no permissions is seen by no one
	require.Empty(t, query(model.NewPermissions(), 1000, 60))
	require.Empty(t, query(nil, 1000, 60))
	require.NotContains(t, query(append(model.NewAnonymousPermissions(), model.MagicGroupIDAuthenticated, groupID), 1000, 60), unreadable.ActivityID)
}

// TestOutbox2_CountByParent verifies that the count applies the same scoping and permission filter as the page
func TestOutbox2_CountByParent(t *testing.T) {

	test := newOutbox2QueryTest(t)

	userID := primitive.NewObjectID()
	groupID := primitive.NewObjectID()
	anonymous := model.NewAnonymousPermissions()

	test.save(t, model.ActorTypeUser, userID, 100, anonymous, nil)
	test.save(t, model.ActorTypeUser, userID, 200, anonymous, nil)
	test.save(t, model.ActorTypeUser, userID, 300, model.Permissions{groupID}, nil)
	test.save(t, model.ActorTypeUser, userID, 400, model.NewPermissions(), nil)
	test.save(t, model.ActorTypeUser, userID, 500, anonymous, func(item *model.OutboxItem) { item.DeleteDate = 999 })
	test.save(t, model.ActorTypeStream, userID, 600, anonymous, nil)

	count := func(permissions model.Permissions) int64 {
		result, err := test.service.CountByParent(test.session, model.ActorTypeUser, userID, permissions)
		require.NoError(t, err)
		return result
	}

	require.Equal(t, int64(2), count(anonymous))
	require.Equal(t, int64(3), count(append(model.NewAnonymousPermissions(), groupID)))
	require.Zero(t, count(model.NewPermissions()))
	require.Zero(t, count(nil))
}

// TestOutbox2_RangeByObjectID verifies that only the actor's own items for that object are returned
func TestOutbox2_RangeByObjectID(t *testing.T) {

	test := newOutbox2QueryTest(t)

	userID := primitive.NewObjectID()
	objectID := "https://example.test/notes/1"
	anonymous := model.NewAnonymousPermissions()
	withObject := func(objectID string, activityType string) func(*model.OutboxItem) {
		return func(item *model.OutboxItem) {
			item.ObjectID = objectID
			item.ActivityType = activityType
		}
	}

	create := test.save(t, model.ActorTypeUser, userID, 100, anonymous, withObject(objectID, "Create"))
	update := test.save(t, model.ActorTypeUser, userID, 200, model.NewPermissions(), withObject(objectID, "Update"))
	test.save(t, model.ActorTypeUser, userID, 300, anonymous, withObject("https://example.test/notes/2", "Create"))
	test.save(t, model.ActorTypeUser, primitive.NewObjectID(), 400, anonymous, withObject(objectID, "Announce"))
	test.save(t, model.ActorTypeStream, userID, 500, anonymous, withObject(objectID, "Create"))
	test.save(t, model.ActorTypeUser, userID, 600, anonymous, func(item *model.OutboxItem) {
		withObject(objectID, "Delete")(item)
		item.DeleteDate = 999
	})

	items, err := test.service.RangeByObjectID(test.session, model.ActorTypeUser, userID, objectID)
	require.NoError(t, err)

	// Lookups for internal use ignore permissions, so the unreadable Update is still found
	require.ElementsMatch(t, []primitive.ObjectID{create.ActivityID, update.ActivityID}, activityIDs(slices.Collect(items)))

	// An unknown object finds nothing
	items, err = test.service.RangeByObjectID(test.session, model.ActorTypeUser, userID, "https://example.test/notes/missing")
	require.NoError(t, err)
	require.Empty(t, slices.Collect(items))
}

// TestOutbox2_RangeByActivityURL verifies that the lookup matches the url field, within the actor's own outbox
func TestOutbox2_RangeByActivityURL(t *testing.T) {

	test := newOutbox2QueryTest(t)

	userID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()
	anonymous := model.NewAnonymousPermissions()

	like := test.save(t, model.ActorTypeUser, userID, 100, anonymous, nil)
	test.save(t, model.ActorTypeUser, userID, 200, anonymous, nil)
	other := test.save(t, model.ActorTypeUser, otherID, 300, anonymous, nil)

	items, err := test.service.RangeByActivityURL(test.session, model.ActorTypeUser, userID, like.URL)
	require.NoError(t, err)
	require.Equal(t, []primitive.ObjectID{like.ActivityID}, activityIDs(slices.Collect(items)))

	// Another actor's activity URL is not in this outbox
	items, err = test.service.RangeByActivityURL(test.session, model.ActorTypeUser, userID, other.URL)
	require.NoError(t, err)
	require.Empty(t, slices.Collect(items))
}
