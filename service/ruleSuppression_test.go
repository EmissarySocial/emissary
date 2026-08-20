package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// suppressionStore is a minimal in-memory data.Collection for RuleSuppression rows. Load ignores
// criteria and returns the first row -- enough to pin Suppress/IsSuppressed behavior, which only
// ever asks "is there one?".
type suppressionStore struct {
	ruleStore
	records []model.RuleSuppression
}

// Load implements the data.Collection interface, backed by this stub's in-memory records
func (s *suppressionStore) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {

	if len(s.records) == 0 {
		return derp.NotFound("test", "empty store")
	}

	*(target.(*model.RuleSuppression)) = s.records[0]
	return nil
}

// Save implements the data.Collection interface, backed by this stub's in-memory records
func (s *suppressionStore) Save(object data.Object, _ string) error {
	s.records = append(s.records, *(object.(*model.RuleSuppression)))
	return nil
}

// TestRuleSuppression pins P7-3's don't-re-import record: deleting an imported entry writes ONE
// suppression (idempotent on repeat), and entries with no remote identity suppress nothing.
func TestRuleSuppression(t *testing.T) {

	store := &suppressionStore{}
	session := ruleSession{store: store}
	service := NewRuleSuppression()

	userID := primitive.NewObjectID()
	followingID := primitive.NewObjectID()
	remoteID := "https://provider.example/mod/rules/123"

	// Nothing is suppressed in an empty store
	suppressed, err := service.IsSuppressed(session, userID, remoteID)
	require.NoError(t, err)
	require.False(t, suppressed)

	// Suppressing writes exactly one record, carrying the full snapshot
	require.NoError(t, service.Suppress(session, userID, followingID, remoteID))
	require.Len(t, store.records, 1)
	require.Equal(t, userID, store.records[0].UserID)
	require.Equal(t, followingID, store.records[0].FollowingID)
	require.Equal(t, remoteID, store.records[0].RemoteID)

	// RULE: suppressing again is a no-op, not a duplicate
	require.NoError(t, service.Suppress(session, userID, followingID, remoteID))
	require.Len(t, store.records, 1)

	// The suppressed entry now reads as suppressed
	suppressed, err = service.IsSuppressed(session, userID, remoteID)
	require.NoError(t, err)
	require.True(t, suppressed)

	// RULE: a locally-created rule (no remote identity) suppresses nothing
	store.records = nil
	require.NoError(t, service.Suppress(session, userID, followingID, ""))
	require.Empty(t, store.records)
}
