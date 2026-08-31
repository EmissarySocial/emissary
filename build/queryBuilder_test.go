package build

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
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
