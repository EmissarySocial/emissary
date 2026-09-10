package upgrades

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// These drive Version31 against a REAL MongoDB, because the behavior that matters is the
// filter's: whether `$in: ["", null]` reaches a row with NO status key at all, as well as one
// with an empty string, while leaving a real status alone. They skip when no database is
// reachable, so `go test ./...` still passes without one.

// insertConnectionWithStatus writes one raw UserConnection document. Pass nil to write a
// record with NO status key, as a row written before the field was ever set would look.
func insertConnectionWithStatus(t *testing.T, database *mongo.Database, status *string) primitive.ObjectID {

	t.Helper()

	id := primitive.NewObjectID()
	document := bson.M{"_id": id}

	if status != nil {
		document["status"] = *status
	}

	_, err := database.Collection("UserConnection").InsertOne(context.Background(), document)
	require.NoError(t, err)

	return id
}

// loadStatus reads back one UserConnection's status
func loadStatus(t *testing.T, database *mongo.Database, id primitive.ObjectID) string {

	t.Helper()

	var result struct {
		Status string `bson:"status"`
	}

	err := database.Collection("UserConnection").FindOne(context.Background(), bson.M{"_id": id}).Decode(&result)
	require.NoError(t, err)

	return result.Status
}

// TestVersion31 confirms both shapes of "no status" become PENDING and a real status survives
func TestVersion31(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	empty := ""
	ready := model.UserConnectionStatusReady

	blank := insertConnectionWithStatus(t, database, &empty)
	missing := insertConnectionWithStatus(t, database, nil)
	configured := insertConnectionWithStatus(t, database, &ready)

	require.NoError(t, Version31(ctx, database))

	require.Equal(t, model.UserConnectionStatusPending, loadStatus(t, database, blank))
	require.Equal(t, model.UserConnectionStatusPending, loadStatus(t, database, missing), "a missing key is the same state as an empty one")
	require.Equal(t, model.UserConnectionStatusReady, loadStatus(t, database, configured), "a set-up connection is left alone")
}

// TestVersion31_Idempotent confirms a second run changes nothing
func TestVersion31_Idempotent(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	empty := ""
	blank := insertConnectionWithStatus(t, database, &empty)

	require.NoError(t, Version31(ctx, database))
	require.NoError(t, Version31(ctx, database))

	require.Equal(t, model.UserConnectionStatusPending, loadStatus(t, database, blank))
}
