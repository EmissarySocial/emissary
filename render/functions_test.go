package render

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFunctions_DollarFormat(t *testing.T) {

	f := FuncMap()

	dollarFormat := f["dollarFormat"].(func(int64) string)

	require.Equal(t, "$12.34", dollarFormat(1234))
}
