package tabular

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTable_HasProblems(t *testing.T) {

	require.False(t, Table{}.HasProblems())
	require.True(t, Table{Problems: []Problem{{Message: "x"}}}.HasProblems())
}
