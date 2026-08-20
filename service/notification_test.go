package service

import (
	"context"
	"testing"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
)

// These tests use a hand-built data.Collection fake (the same approach as collection_test.go and
// collectionItem_test.go) because PurgeBefore's entire contract IS the criteria it hands to
// HardDelete.  The fake captures that expression so the test can assert on its shape directly.

/******************************************
 * purgeStore -- an in-memory data.Collection that records HardDelete's criteria.
 ******************************************/

// purgeStore is an in-memory data.Collection that records the criteria HardDelete was called with
type purgeStore struct {
	criteria exp.Expression
	called   bool
	err      error
}

// Context implements the interface, returning a background context
func (c *purgeStore) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *purgeStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *purgeStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *purgeStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load implements the data.Collection interface. Unused by these tests.
func (c *purgeStore) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *purgeStore) Save(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// Delete implements the data.Collection interface. Unused by these tests.
func (c *purgeStore) Delete(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *purgeStore) HardDelete(criteria exp.Expression) error {
	c.called = true
	c.criteria = criteria
	return c.err
}

// purgeSession is a data.Session that hands out a single purgeStore
type purgeSession struct {
	store data.Collection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s purgeSession) Collection(string) data.Collection { return s.store }

// Context implements the interface, returning a background context
func (s purgeSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s purgeSession) Close() {}

// predicatesOf flattens every Predicate in an Expression into a slice.  The MatcherFunc returns
// TRUE so the walk visits the whole tree -- AndExpression.Match short-circuits on FALSE.
func predicatesOf(criteria exp.Expression) []exp.Predicate {

	result := make([]exp.Predicate, 0)

	criteria.Match(func(predicate exp.Predicate) bool {
		result = append(result, predicate)
		return true
	})

	return result
}

/******************************************
 * Tests
 ******************************************/

// TestNotification_PurgeBefore confirms that the purge selects on createDate alone.  The absent
// readDate predicate is the point: retention is uniform, so an unread notification must age out
// on the same clock as a read one.
func TestNotification_PurgeBefore(t *testing.T) {

	const cutoff = int64(1_234_567_890)

	store := &purgeStore{}
	service := NewNotification()

	err := service.PurgeBefore(purgeSession{store: store}, cutoff)

	require.Nil(t, err)
	require.True(t, store.called)

	predicates := predicatesOf(store.criteria)

	require.Len(t, predicates, 1)
	require.Equal(t, "createDate", predicates[0].Field)
	require.Equal(t, exp.OperatorLessThan, predicates[0].Operator)
	require.Equal(t, cutoff, predicates[0].Value)

	// Regression guard: read-state must never re-enter the criteria.
	for _, predicate := range predicates {
		require.NotEqual(t, "readDate", predicate.Field)
	}
}

// TestNotification_PurgeBefore_Error confirms that a storage failure is reported, not swallowed.
func TestNotification_PurgeBefore_Error(t *testing.T) {

	store := &purgeStore{err: derp.Internal("test", "database is on fire")}
	service := NewNotification()

	err := service.PurgeBefore(purgeSession{store: store}, 1_234_567_890)

	require.NotNil(t, err)
	require.Equal(t, "database is on fire", derp.Message(derp.RootCause(err)))
}
