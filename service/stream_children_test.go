package service

import (
	"context"
	"testing"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests pin the query-side half of the /pub/children access-control fix
// (see handler/activitypub_stream/children.go): RangeOutboxByParent must test
// the outbox flag in the QUERY so children that followers were never sent never
// leave the database, while per-viewer permissions stay out of the query. They use
// a fake data.Collection that captures the criteria RangeOutboxByParent builds,
// then walk it with exp.Expression.Match to assert which predicates are present.

/******************************************
 * criteriaCollection — a data.Collection that records the criteria passed to
 * Iterator and returns an empty iterator. It implements only what service.Range
 * exercises; every other method is an explicit "unused".
 ******************************************/

// criteriaCollection is a data.Collection that records the criteria it was queried with, and returns nothing
type criteriaCollection struct {
	captured exp.Expression
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *criteriaCollection) Iterator(criteria exp.Expression, _ ...option.Option) (data.Iterator, error) {
	c.captured = criteria
	return &emptyIterator{}, nil
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *criteriaCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

// Load implements the data.Collection interface. Unused by these tests.
func (c *criteriaCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *criteriaCollection) Save(data.Object, string) error { return derp.NotFound("test", "unused") }

// Delete implements the data.Collection interface. Unused by these tests.
func (c *criteriaCollection) Delete(data.Object, string) error {
	return derp.NotFound("test", "unused")
}

// Count implements the data.Collection interface. Unused by these tests.
func (c *criteriaCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.NotFound("test", "unused")
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *criteriaCollection) HardDelete(exp.Expression) error { return derp.NotFound("test", "unused") }

// Context implements the interface, returning a background context
func (c *criteriaCollection) Context() context.Context { return context.Background() }

// emptyIterator is a data.Iterator with no records.
type emptyIterator struct{}

// Next implements the data.Iterator interface, reporting that there are no more records
func (i *emptyIterator) Next(any) bool { return false }

// Count implements the data.Iterator interface, reporting an empty result set
func (i *emptyIterator) Count() int { return 0 }

// Error implements the data.Iterator interface. The stub never fails.
func (i *emptyIterator) Error() error { return nil }

// Close implements the data.Iterator interface. The stub holds no resources to release.
func (i *emptyIterator) Close() error { return nil }

// criteriaSession hands every Collection() call the same capturing collection.
type criteriaSession struct {
	collection *criteriaCollection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s criteriaSession) Collection(string) data.Collection { return s.collection }

// Context implements the interface, returning a background context
func (s criteriaSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s criteriaSession) Close() {}

// collectPredicates walks an Expression and returns every Predicate it contains.
func collectPredicates(criteria exp.Expression) []exp.Predicate {
	result := make([]exp.Predicate, 0)
	criteria.Match(func(predicate exp.Predicate) bool {
		result = append(result, predicate)
		return true // AndExpression.Match short-circuits on false, so return true to visit every predicate
	})
	return result
}

// hasPredicate returns TRUE if the slice contains a Predicate on the given field with the given operator.
func hasPredicate(predicates []exp.Predicate, field string, operator string) bool {
	for _, predicate := range predicates {
		if (predicate.Field == field) && (predicate.Operator == operator) {
			return true
		}
	}
	return false
}

// TestStream_RangeOutboxByParent_Criteria verifies the exact query that RangeOutboxByParent issues
func TestStream_RangeOutboxByParent_Criteria(t *testing.T) {

	parentID := primitive.NewObjectID()

	collection := &criteriaCollection{}
	session := criteriaSession{collection: collection}
	service := &Stream{}

	iterator, err := service.RangeOutboxByParent(session, parentID)

	require.Nil(t, err)
	require.NotNil(t, iterator)

	// Draining the iter.Seq forces service.Range to build the underlying data.Iterator,
	// which is what hands our fake collection the criteria.
	for range iterator {
		t.Fatal("empty iterator must yield no streams")
	}

	predicates := collectPredicates(collection.captured)

	// RULE: The query must scope to the requested parent.
	require.True(t, hasPredicate(predicates, "parentId", exp.OperatorEqual), "must filter by parentId")

	// RULE: The outbox flag must be in the QUERY, so a child that followers were never sent
	// does not leave the database
	require.True(t, hasPredicate(predicates, "isOutbox", exp.OperatorEqual), "must filter by the outbox flag")

	for _, predicate := range predicates {
		if predicate.Field == "isOutbox" {
			require.Equal(t, true, predicate.Value)
		}

		// RULE: Nothing filters on a publish date any more (emissary-specs PUBLICATION §1.12)
		require.NotEqual(t, "publishDate", predicate.Field)
		require.NotEqual(t, "unpublishDate", predicate.Field)
	}

	// RULE: service.List always excludes soft-deleted records.
	require.True(t, hasPredicate(predicates, "deleteDate", exp.OperatorEqual), "must exclude deleted streams")

	// RULE: Permissions are intentionally NOT part of the query -- per-viewer gating is
	// left to the handler (Stream.IsVisibleTo). Guard against a regression that pushes
	// a "permissions"/"defaultAllow" filter down into the query.
	require.False(t, hasPredicate(predicates, "permissions", exp.OperatorIn), "permissions must not be in the query")
	require.False(t, hasPredicate(predicates, "defaultAllow", exp.OperatorIn), "defaultAllow must not be in the query")
}
