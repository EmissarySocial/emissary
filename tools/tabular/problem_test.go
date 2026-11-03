package tabular

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProblem_String(t *testing.T) {

	require.Equal(t, `line 4, column "label": Value is too long`, Problem{Line: 4, Column: "label", Message: "Value is too long"}.String())
	require.Equal(t, "line 2: Entry must be an object", Problem{Line: 2, Message: "Entry must be an object"}.String())
	require.Equal(t, `line 1, column "a\"b": x`, Problem{Line: 1, Column: `a"b`, Message: "x"}.String())
}
