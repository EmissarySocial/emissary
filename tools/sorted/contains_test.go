package sorted

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestContainsAll_SimpleSuccess(t *testing.T) {

	subset := []string{"A", "C", "E", "G"}
	superset := []string{"A", "B", "C", "D", "E", "F", "G"}

	require.True(t, ContainsAll(subset, superset))
}

func TestContainsAll_SimpleFailure(t *testing.T) {

	subset := []string{"A", "C", "D", "E", "G"}
	superset := []string{"A", "B", "C", "E", "F", "G"}

	require.False(t, ContainsAll(subset, superset))
}

func TestContainsAll_FailBegin(t *testing.T) {

	subset := []string{"A", "C", "E", "G"}
	superset := []string{"B", "C", "D", "E", "F", "G"}

	require.False(t, ContainsAll(subset, superset))
}

func TestContainsAll_FailEnd(t *testing.T) {

	subset := []string{"A", "C", "E", "G"}
	superset := []string{"A", "B", "C", "D", "E"}

	require.False(t, ContainsAll(subset, superset))
}
