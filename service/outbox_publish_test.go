package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestPermissionsToHex pins the permission (de)serialization used to carry Outbox.Publish
// permissions across the "Outbox-Publish" task boundary. ObjectIDs don't survive task storage
// round-trips reliably, so Publish serializes them to hex and consumer.OutboxPublish re-parses
// them; a drift here would silently deliver activities to followers who lack view permission,
// or drop delivery to followers who have it. See POST-COMMIT-FEDERATION.md §5 / F2.
func TestPermissionsToHex(t *testing.T) {

	// Empty / nil serialize to an empty (non-nil) slice.
	require.Equal(t, []string{}, permissionsToHex(nil))
	require.Equal(t, []string{}, permissionsToHex(model.Permissions{}))

	a, err := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
	require.Nil(t, err)

	b, err := primitive.ObjectIDFromHex("5f2b8a9c1d4e6f0011223344")
	require.Nil(t, err)

	// Serialization preserves order and values.
	hexes := permissionsToHex(model.Permissions{a, b})
	require.Equal(t, []string{"507f1f77bcf86cd799439011", "5f2b8a9c1d4e6f0011223344"}, hexes)

	// Full round-trip back through the consumer-side parse (mirrors consumer.OutboxPublish).
	parsed := make(model.Permissions, 0)
	for _, hex := range hexes {
		id, err := primitive.ObjectIDFromHex(hex)
		require.Nil(t, err)
		parsed = append(parsed, id)
	}

	require.Equal(t, model.Permissions{a, b}, parsed)
}
