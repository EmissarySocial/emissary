package mastodon

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestMarkHomeRead_IgnoresNonNewsItemIDs confirms a marker that names no NewsItem marks nothing and never fails.
// The nil service proves no lookup is attempted: reaching it would panic.
func TestMarkHomeRead_IgnoresNonNewsItemIDs(t *testing.T) {

	userID := primitive.NewObjectID()

	for _, lastReadID := range []string{"12345", "p_aHR0cHM6Ly9leGFtcGxlLmNvbS8x", "not-an-id"} {
		require.NoError(t, markHomeRead(nil, nil, userID, lastReadID), lastReadID)
	}
}
