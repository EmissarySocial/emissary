package service

import (
	"context"
	"testing"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests pin the query-side half of the /pub/children access-control fix
// (see handler/activitypub_stream/children.go): RangePublishedByParent must apply
// the publish-date window in the QUERY so unpublished/scheduled children never
// leave the database, while per-viewer permissions stay out of the query. They use
// a fake data.Collection that captures the criteria RangePublishedByParent builds,
// then walk it with exp.Expression.Match to assert which predicates are present.

/******************************************
 * criteriaCollection — a data.Collection that records the criteria passed to
 * Iterator and returns an empty iterator. It implements only what service.Range
 * exercises; every other method is an explicit "unused".
 ******************************************/

type criteriaCollection struct {
	captured exp.Expression
}

func (c *criteriaCollection) Iterator(criteria exp.Expression, _ ...option.Option) (data.Iterator, error) {
	c.captured = criteria
	return &emptyIterator{}, nil
}

func (c *criteriaCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}
func (c *criteriaCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.NotFound("test", "unused")
}
func (c *criteriaCollection) Save(data.Object, string) error { return derp.NotFound("test", "unused") }
func (c *criteriaCollection) Delete(data.Object, string) error {
	return derp.NotFound("test", "unused")
}
func (c *criteriaCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.NotFound("test", "unused")
}
func (c *criteriaCollection) HardDelete(exp.Expression) error { return derp.NotFound("test", "unused") }
func (c *criteriaCollection) Context() context.Context        { return context.Background() }

// emptyIterator is a data.Iterator with no records.
type emptyIterator struct{}

func (i *emptyIterator) Next(any) bool { return false }
func (i *emptyIterator) Count() int    { return 0 }
func (i *emptyIterator) Error() error  { return nil }
func (i *emptyIterator) Close() error  { return nil }

// criteriaSession hands every Collection() call the same capturing collection.
type criteriaSession struct {
	collection *criteriaCollection
}

func (s criteriaSession) Collection(string) data.Collection { return s.collection }
func (s criteriaSession) Context() context.Context          { return context.Background() }
func (s criteriaSession) Close()                            {}

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

func TestStream_RangePublishedByParent_Criteria(t *testing.T) {

	parentID := primitive.NewObjectID()

	collection := &criteriaCollection{}
	session := criteriaSession{collection: collection}
	service := &Stream{}

	before := time.Now().Unix()
	iterator, err := service.RangePublishedByParent(session, parentID)
	after := time.Now().Unix()

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

	// RULE: The publish-date window (publishDate <= now <= unpublishDate) must be in the QUERY,
	// so unpublished/scheduled children never leave the database.
	require.True(t, hasPredicate(predicates, "publishDate", exp.OperatorLessOrEqual), "must bound publishDate")
	require.True(t, hasPredicate(predicates, "unpublishDate", exp.OperatorGreaterOrEqual), "must bound unpublishDate")

	// RULE: service.List always excludes soft-deleted records.
	require.True(t, hasPredicate(predicates, "deleteDate", exp.OperatorEqual), "must exclude deleted streams")

	// RULE: The "now" bound must be a sane timestamp captured at call time.
	for _, predicate := range predicates {
		if predicate.Field == "publishDate" {
			now, ok := predicate.Value.(int64)
			require.True(t, ok, "publishDate bound must be an int64 unix timestamp")
			require.GreaterOrEqual(t, now, before)
			require.LessOrEqual(t, now, after)
		}
	}

	// RULE: Permissions are intentionally NOT part of the query -- per-viewer gating is
	// left to the handler (Stream.IsVisibleTo). Guard against a regression that pushes
	// a "permissions"/"defaultAllow" filter down into the query.
	require.False(t, hasPredicate(predicates, "permissions", exp.OperatorIn), "permissions must not be in the query")
	require.False(t, hasPredicate(predicates, "defaultAllow", exp.OperatorIn), "defaultAllow must not be in the query")
}
