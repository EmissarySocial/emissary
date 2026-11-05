package build

import (
	"reflect"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data/option"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
)

// bsonName returns the stored name of a struct field, without its bson options
func bsonName(t *testing.T, structType reflect.Type, fieldName string) string {
	t.Helper()

	field, ok := structType.FieldByName(fieldName)
	require.True(t, ok, "%s has no field %s", structType.Name(), fieldName)

	name, _, _ := strings.Cut(field.Tag.Get("bson"), ",")
	require.NotEmpty(t, name, "%s.%s has no bson name", structType.Name(), fieldName)
	return name
}

// TestQueryBuilder_Tags_FiltersTheStoredField verifies that Tags filters on the names that
// StreamSummary and Tag actually write, because MongoDB matches field names by exact case.
func TestQueryBuilder_Tags_FiltersTheStoredField(t *testing.T) {

	field := bsonName(t, reflect.TypeFor[model.StreamSummary](), "Tags") + "." + bsonName(t, reflect.TypeFor[model.Tag](), "Name")
	require.Equal(t, "tags.name", field)

	builder := NewEmptyQueryBuilder[model.StreamSummary]().Tags("Sunset", "travel")
	require.Equal(t, exp.In(field, []string{"Sunset", "travel"}), builder.criteria)
}

// TestQueryBuilder_Tags_NoTags verifies that Tags with no names still filters, matching nothing
func TestQueryBuilder_Tags_NoTags(t *testing.T) {

	builder := NewEmptyQueryBuilder[model.StreamSummary]().Tags()
	require.Equal(t, exp.In("tags.name", []string(nil)), builder.criteria)
}

// TestQueryBuilder_CaseInsensitive verifies that CaseInsensitive asks the database to ignore case,
// and that a builder carries no case option until one is chosen
func TestQueryBuilder_CaseInsensitive(t *testing.T) {

	builder := NewEmptyQueryBuilder[model.StreamSummary]()
	require.False(t, builder.caseSensitive.IsPresent())

	require.Contains(t, builder.CaseInsensitive().makeOptions(), option.CaseSensitive(false))
	require.Contains(t, builder.CaseSensitive().makeOptions(), option.CaseSensitive(true))
}

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
