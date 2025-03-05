package sorted

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
