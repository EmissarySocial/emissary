package build

import (
	"reflect"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
)

// TestEmptyQueryBuilder_Slice verifies that an empty QueryBuilder returns an empty (not nil)
// slice without dereferencing the service or session it does not carry.
func TestEmptyQueryBuilder_Slice(t *testing.T) {

	builder := NewEmptyQueryBuilder[model.StreamSummary]()

	result, err := builder.Slice()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result)
}

// TestEmptyQueryBuilder_Count verifies that an empty QueryBuilder counts zero
func TestEmptyQueryBuilder_Count(t *testing.T) {

	builder := NewEmptyQueryBuilder[model.StreamSummary]()

	result, err := builder.Count()

	require.NoError(t, err)
	require.Zero(t, result)
}

// TestEmptyQueryBuilder_StaysEmptyWhenChained verifies that the chainable methods, which are the
// only API a Template can reach, cannot talk an empty QueryBuilder into querying the database.
func TestEmptyQueryBuilder_StaysEmptyWhenChained(t *testing.T) {

	builder := NewEmptyQueryBuilder[model.StreamSummary]().
		Top12().
		ByRank().
		Reverse().
		Where("templateId", "folder").
		WhereGT("rank", 0).
		WhereIN("stateId", []string{"published"}).
		WhereBeginsWith("label", "A").
		WhereContains("label", "B").
		Tags("example").
		Indexable().
		Featured().
		CaseSensitive()

	result, err := builder.Slice()

	require.NoError(t, err)
	require.Empty(t, result)

	count, err := builder.Count()

	require.NoError(t, err)
	require.Zero(t, count)
}

// TestQueryBuilder_Tags_FiltersTheStoredField verifies that the tag filter names the field a Stream's
// tags are stored under, which MongoDB matches by case
func TestQueryBuilder_Tags_FiltersTheStoredField(t *testing.T) {

	// Read the stored names from the bson tags, so a rename on either side fails here
	bsonName := func(structType reflect.Type, fieldName string) string {
		field, ok := structType.FieldByName(fieldName)
		require.True(t, ok, fieldName)
		name, _, _ := strings.Cut(field.Tag.Get("bson"), ",")
		return name
	}

	stored := bsonName(reflect.TypeFor[model.StreamSummary](), "Tags") + "." + bsonName(reflect.TypeFor[model.Tag](), "Name")
	require.Equal(t, bsonName(reflect.TypeFor[model.Stream](), "Tags"), bsonName(reflect.TypeFor[model.StreamSummary](), "Tags"))

	builder := NewEmptyQueryBuilder[model.StreamSummary]().Tags("travel", "food")

	predicate, isPredicate := builder.criteria.(exp.Predicate)
	require.True(t, isPredicate)
	require.Equal(t, stored, predicate.Field)
	require.Equal(t, exp.OperatorIn, predicate.Operator)
	require.Equal(t, []string{"travel", "food"}, predicate.Value)
}
