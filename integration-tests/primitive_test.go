package integration

import (
	"testing"

	"github.com/benpate/rosetta/convert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// This test confirms that convert.SliceOfMap works with one level
// of BSON primitive.A, primitive.M structures.
// NOTE: It WON'T work if you put another BSON structure
// inside one of these maps, because map values are passed through
// without being converted.
func Test_ConvertPrimitive(t *testing.T) {

	value := primitive.A{
		primitive.M{
			"one":   1,
			"two":   2,
			"three": 3,
		},
		primitive.M{
			"five":  5,
			"six":   6,
			"seven": 7,
		},
	}

	expected := []map[string]any{
		{
			"one":   1,
			"two":   2,
			"three": 3,
		},
		{
			"five":  5,
			"six":   6,
			"seven": 7,
		},
	}

	actual := convert.SliceOfMap(value)
	require.Equal(t, expected, actual)
}
