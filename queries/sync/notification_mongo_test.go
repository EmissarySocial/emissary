package sync

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestNotificationIndex_UnreadByDate verifies that the hourly unread scan, which names no User,
// is answered from idx_Notification_UnreadByDate and not by reading the collection
func TestNotificationIndex_UnreadByDate(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Notification(ctx, database))

	// A few rows, so that the planner has something to choose between
	for index := range 20 {
		_, err := database.Collection("Notification").InsertOne(ctx, bson.M{
			"_id":        primitive.NewObjectID(),
			"userId":     primitive.NewObjectID(),
			"createDate": int64(index),
			"readDate":   int64(math.MaxInt64),
			"deleteDate": 0,
		})
		require.NoError(t, err)
	}

	// The same filter that queries.NotificationUsersWithUnread sends
	command := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "distinct", Value: "Notification"},
			{Key: "key", Value: "userId"},
			{Key: "query", Value: bson.M{
				"deleteDate": 0,
				"readDate":   int64(math.MaxInt64),
				"createDate": bson.M{"$lte": int64(10)},
			}},
		}},
		{Key: "verbosity", Value: "queryPlanner"},
	}

	plan := bson.M{}
	require.NoError(t, database.RunCommand(ctx, command).Decode(&plan))

	planJSON, err := bson.MarshalExtJSON(plan["queryPlanner"], false, false)
	require.NoError(t, err)

	winningPlan, err := bson.MarshalExtJSON(plan["queryPlanner"].(bson.M)["winningPlan"], false, false)
	require.NoError(t, err)

	require.Contains(t, string(winningPlan), "idx_Notification_UnreadByDate", string(planJSON))
	require.NotContains(t, string(winningPlan), "COLLSCAN")
}
