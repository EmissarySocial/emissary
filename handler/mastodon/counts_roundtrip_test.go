package mastodon

import (
	"testing"
	"time"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestApplyDocumentCounts_SurvivesTheDatabase stores normalized totals the way the cache does and reads them back,
// because a total that comes back as a 32-bit integer reads as zero.
func TestApplyDocumentCounts_SurvivesTheDatabase(t *testing.T) {

	stored, err := bson.Marshal(map[string]any{
		"type":    "Note",
		"id":      "https://example.com/n/1",
		"likes":   map[string]any{"totalItems": int64(7)},
		"shares":  map[string]any{"totalItems": int64(3)},
		"updated": time.Date(2026, 10, 1, 12, 30, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	loaded := mapof.NewAny()
	require.NoError(t, bson.Unmarshal(stored, &loaded))

	status := newTestStatus()
	applyDocumentCounts(&status, streams.NewDocument(loaded))

	require.Equal(t, 7, status.FavouritesCount)
	require.Equal(t, 3, status.ReblogsCount)
	require.Equal(t, "2026-10-01T12:30:00.000Z", status.EditedAt)
}

// newTestStatus returns an empty Status.
func newTestStatus() object.Status {
	return object.Status{}
}

// TestActorFlags_SurviveTheDatabase stores a normalized actor the way the cache does and reads its join date
// and approval setting back, as the account mapping does.
func TestActorFlags_SurviveTheDatabase(t *testing.T) {

	stored, err := bson.Marshal(map[string]any{
		"type":                      "Person",
		"id":                        "https://example.com/users/ben",
		"published":                 time.Date(2020, 8, 27, 0, 0, 0, 0, time.UTC),
		"manuallyApprovesFollowers": true,
	})
	require.NoError(t, err)

	loaded := mapof.NewAny()
	require.NoError(t, bson.Unmarshal(stored, &loaded))

	document := streams.NewDocument(loaded)

	require.Equal(t, 2020, document.Published().Year())
	require.True(t, document.Get("manuallyApprovesFollowers").Bool())
}
