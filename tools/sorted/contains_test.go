package sorted

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestContains verifies that Contains finds present values and rejects absent ones
func TestContains(t *testing.T) {

	set := []string{"A", "C", "E", "G"}

	require.True(t, Contains(set, "A"))
	require.False(t, Contains(set, "B"))
	require.True(t, Contains(set, "C"))
	require.False(t, Contains(set, "D"))
	require.True(t, Contains(set, "E"))
	require.False(t, Contains(set, "F"))
	require.True(t, Contains(set, "G"))
	require.False(t, Contains(set, "H"))
}

// TestContainsAll_SimpleSuccess verifies that a subset fully contained in the superset returns TRUE
func TestContainsAll_SimpleSuccess(t *testing.T) {

	subset := []string{"A", "C", "E", "G"}
	superset := []string{"A", "B", "C", "D", "E", "F", "G"}

	require.True(t, ContainsAll(subset, superset))
}

// TestContainsAll_SimpleFailure verifies that a value missing from the middle of the superset returns FALSE
func TestContainsAll_SimpleFailure(t *testing.T) {

	subset := []string{"A", "C", "D", "E", "G"}
	superset := []string{"A", "B", "C", "E", "F", "G"}

	require.False(t, ContainsAll(subset, superset))
}

// TestContainsAll_FailBegin verifies that a value missing from the start of the superset returns FALSE
func TestContainsAll_FailBegin(t *testing.T) {

	subset := []string{"A", "C", "E", "G"}
	superset := []string{"B", "C", "D", "E", "F", "G"}

	require.False(t, ContainsAll(subset, superset))
}

// TestContainsAll_FailEnd verifies that a value missing from the end of the superset returns FALSE
func TestContainsAll_FailEnd(t *testing.T) {

	subset := []string{"A", "C", "E", "G"}
	superset := []string{"A", "B", "C", "D", "E"}

	require.False(t, ContainsAll(subset, superset))
}
