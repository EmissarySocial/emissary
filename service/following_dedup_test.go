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

// These tests cover Following.reconcileAndSave / reconcileDuplicate -- the concurrency dedup that
// makes duplicate follows of one actor converge on a single row. They use an injected-conflict
// fake data.Collection (the same technique as collectionItem_test.go's SaveUnique tests), because
// the real guarantee -- the unique index idx_Following_User_Profile_Unique -- lives in MongoDB and
// surfaces to the service only as a derp.Conflict on Save.

/******************************************
 * In-Memory Fake
 ******************************************/

// followingLoad is one scripted result for the fake collection's Load, matched by call order.
type followingLoad struct {
	record model.Following
	found  bool
}

// followingStore is a data.Collection fake whose Load results are scripted by call index and whose
// Save can inject a duplicate-key Conflict on its first call, exactly as the unique index would when
// a competing writer inserts the same (userId, profileUrl) first.
type followingStore struct {
	loads     []followingLoad // Load results, indexed by call order
	loadCalls int
	saveErr   error // when set, the FIRST Save returns this (the optimistic insert losing the race)
	saveCalls int
	saved     []model.Following
	deleted   []primitive.ObjectID
}

// Context implements the interface, returning a background context
func (s *followingStore) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (s *followingStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (s *followingStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (s *followingStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

// Load returns the scripted result for this call index, or NotFound once the script is exhausted.
func (s *followingStore) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {

	index := s.loadCalls
	s.loadCalls++

	if index < len(s.loads) && s.loads[index].found {
		if following, ok := target.(*model.Following); ok {
			*following = s.loads[index].record
			return nil
		}
	}

	return derp.NotFound("test", "not found")
}

// Save records the object, but fails the FIRST call with the injected error (the lost race).
func (s *followingStore) Save(object data.Object, _ string) error {

	s.saveCalls++

	if s.saveErr != nil && s.saveCalls == 1 {
		return s.saveErr
	}

	if following, ok := object.(*model.Following); ok {
		s.saved = append(s.saved, *following)
	}

	return nil
}

// Delete implements the data.Collection interface. Unused by these tests.
func (s *followingStore) Delete(object data.Object, _ string) error {

	if following, ok := object.(*model.Following); ok {
		s.deleted = append(s.deleted, following.FollowingID)
	}

	return nil
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (s *followingStore) HardDelete(exp.Expression) error { return nil }

// followingSession is a data.Session whose Collection always returns the fake store.
type followingSession struct {
	data.Session
	store data.Collection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s followingSession) Collection(string) data.Collection { return s.store }

/******************************************
 * reconcileAndSave -- the retry
 ******************************************/

// When the optimistic insert loses the race, reconcileAndSave folds onto the winner and updates in
// place -- one failed insert, one successful update -- instead of erroring or double-inserting.
func TestFollowing_reconcileAndSave_DuplicateKeyFoldsOntoWinner(t *testing.T) {

	userID := primitive.NewObjectID()

	winner := model.NewFollowing()
	winner.FollowingID = primitive.NewObjectID()
	winner.UserID = userID
	winner.ProfileURL = "https://example.test/users/gargron"
	winner.CreateDate = 100

	store := &followingStore{
		saveErr: duplicateKeyError(),
		loads: []followingLoad{
			{found: false},                // reconcileDuplicate #1: no duplicate visible yet
			{found: true, record: winner}, // reconcileDuplicate #2 (post-conflict): winner is visible
		},
	}

	service := NewFollowing()
	session := followingSession{store: store}

	following := model.NewFollowing()
	following.UserID = userID
	following.ProfileURL = "https://example.test/users/gargron"

	err := (&service).reconcileAndSave(session, &following, func() error {
		return store.Save(&following, "loser")
	})

	require.Nil(t, err)
	require.Equal(t, winner.FollowingID, following.FollowingID) // folded onto the winner
	require.Equal(t, 2, store.saveCalls)                        // one failed insert, one successful update
	require.Empty(t, store.deleted)                             // an in-memory loser has nothing to clean up
}

// A non-Conflict Save error propagates unchanged and is NOT retried.
func TestFollowing_reconcileAndSave_SaveErrorPropagates(t *testing.T) {

	store := &followingStore{
		saveErr: derp.Internal("test", "disk on fire"),
		loads:   []followingLoad{{found: false}},
	}

	service := NewFollowing()
	session := followingSession{store: store}

	following := model.NewFollowing()
	following.UserID = primitive.NewObjectID()
	following.ProfileURL = "https://example.test/users/gargron"

	err := (&service).reconcileAndSave(session, &following, func() error {
		return store.Save(&following, "x")
	})

	require.NotNil(t, err)
	require.False(t, derp.IsConflict(err))
	require.Equal(t, 1, store.saveCalls) // failed once, not retried
}

/******************************************
 * reconcileDuplicate -- the fold
 ******************************************/

// An unresolved ProfileURL collides with nothing (it is excluded from the unique index too), so
// reconcileDuplicate is a no-op and never touches the database.
func TestFollowing_reconcileDuplicate_UnresolvedProfileURLIsNoop(t *testing.T) {

	store := &followingStore{}
	service := NewFollowing()
	session := followingSession{store: store}

	following := model.NewFollowing()
	originalID := following.FollowingID
	following.UserID = primitive.NewObjectID()
	following.ProfileURL = "" // not yet resolved

	err := (&service).reconcileDuplicate(session, &following)

	require.Nil(t, err)
	require.Equal(t, originalID, following.FollowingID) // unchanged
	require.Equal(t, 0, store.loadCalls)                // never queried
}

// With no existing follow of this actor, reconcileDuplicate leaves the record's identity alone.
func TestFollowing_reconcileDuplicate_NoDuplicateLeavesIdentity(t *testing.T) {

	store := &followingStore{loads: []followingLoad{{found: false}}}
	service := NewFollowing()
	session := followingSession{store: store}

	following := model.NewFollowing()
	originalID := following.FollowingID
	following.UserID = primitive.NewObjectID()
	following.ProfileURL = "https://example.test/users/gargron"

	err := (&service).reconcileDuplicate(session, &following)

	require.Nil(t, err)
	require.Equal(t, originalID, following.FollowingID) // unchanged
	require.Empty(t, store.deleted)
}

// The create path inserts a row before Connect resolves profileUrl. If that resolved key already
// belongs to another follow, reconcileDuplicate retires the pre-inserted (now-orphan) row and adopts
// the winner -- so no empty-profileUrl leftover survives.
func TestFollowing_reconcileDuplicate_PersistedOrphanIsCleanedUp(t *testing.T) {

	userID := primitive.NewObjectID()
	staleID := primitive.NewObjectID()
	winnerID := primitive.NewObjectID()

	winner := model.NewFollowing()
	winner.FollowingID = winnerID
	winner.UserID = userID
	winner.ProfileURL = "https://example.test/users/gargron"
	winner.CreateDate = 100

	stale := model.NewFollowing()
	stale.FollowingID = staleID
	stale.UserID = userID
	stale.ProfileURL = "https://example.test/users/gargron"
	stale.CreateDate = 50

	store := &followingStore{
		loads: []followingLoad{
			{found: true, record: winner}, // Load #1: the duplicate lookup finds the winner
			{found: true, record: stale},  // Load #2: LoadByID resolves the stale row to delete
		},
	}

	service := NewFollowing()
	session := followingSession{store: store}

	// The incoming record was already persisted under its own id (CreateDate != 0 => not IsNew).
	following := model.NewFollowing()
	following.FollowingID = staleID
	following.UserID = userID
	following.ProfileURL = "https://example.test/users/gargron"
	following.CreateDate = 50

	err := (&service).reconcileDuplicate(session, &following)

	require.Nil(t, err)
	require.Equal(t, winnerID, following.FollowingID)              // folded onto the winner
	require.Equal(t, []primitive.ObjectID{staleID}, store.deleted) // orphan retired
}
