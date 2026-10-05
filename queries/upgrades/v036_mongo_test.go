package upgrades

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// loadDomainCreateDate returns the createDate of the singleton Domain record.
func loadDomainCreateDate(t *testing.T, database *mongo.Database) int64 {

	t.Helper()

	var result struct {
		CreateDate int64 `bson:"createDate"`
	}

	err := database.Collection("Domain").FindOne(context.Background(), bson.M{"_id": primitive.NilObjectID}).Decode(&result)
	require.NoError(t, err)

	return result.CreateDate
}

func TestVersion36_MissingCreateDate(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	_, err := database.Collection("Domain").InsertOne(ctx, bson.M{"_id": primitive.NilObjectID, "label": "Stripped"})
	require.NoError(t, err)

	require.NoError(t, Version36(ctx, database))
	require.NotZero(t, loadDomainCreateDate(t, database), "a stripped journal must be restored")
}

func TestVersion36_ZeroCreateDate(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	_, err := database.Collection("Domain").InsertOne(ctx, bson.M{"_id": primitive.NilObjectID, "createDate": int64(0)})
	require.NoError(t, err)

	require.NoError(t, Version36(ctx, database))
	require.NotZero(t, loadDomainCreateDate(t, database), "a zero createDate must be restored")
}

func TestVersion36_HealthyRecordUntouched(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	_, err := database.Collection("Domain").InsertOne(ctx, bson.M{"_id": primitive.NilObjectID, "createDate": int64(12345)})
	require.NoError(t, err)

	require.NoError(t, Version36(ctx, database))
	require.NoError(t, Version36(ctx, database))
	require.Equal(t, int64(12345), loadDomainCreateDate(t, database), "an intact createDate is never rewritten")
}

func TestVersion25_KeepsJournal(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	_, err := database.Collection("Domain").InsertOne(ctx, bson.M{"_id": primitive.NilObjectID, "createDate": int64(12345)})
	require.NoError(t, err)

	_, err = database.Collection("Connection").InsertOne(ctx, bson.M{"_id": primitive.NewObjectID(), "providerId": "STRIPE"})
	require.NoError(t, err)

	require.NoError(t, Version25(ctx, database))
	require.Equal(t, int64(12345), loadDomainCreateDate(t, database), "copying connections must not strip the journal")

	var result struct {
		Connections map[string]any `bson:"connections"`
	}

	require.NoError(t, database.Collection("Domain").FindOne(ctx, bson.M{}).Decode(&result))
	require.Contains(t, result.Connections, "STRIPE")
}
