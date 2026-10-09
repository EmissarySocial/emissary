package upgrades

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func insertStreamWithContext(t *testing.T, database *mongo.Database, url string, contextValue *string) primitive.ObjectID {

	t.Helper()

	id := primitive.NewObjectID()
	document := bson.M{
		"_id":     id,
		"url":     url,
		"context": "",
	}

	if contextValue != nil {
		document["context"] = *contextValue
	}

	_, err := database.Collection("Stream").InsertOne(context.Background(), document)
	require.NoError(t, err)

	return id
}

func loadStreamContext(t *testing.T, database *mongo.Database, id primitive.ObjectID) string {

	t.Helper()

	var result struct {
		Context string `bson:"context"`
	}

	err := database.Collection("Stream").FindOne(context.Background(), bson.M{"_id": id}).Decode(&result)
	require.NoError(t, err)

	return result.Context
}

func TestVersion33(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	legacyURL := "https://example.com/stream/legacy"
	legacyContext := legacyURL + "/pub/context"
	validContext := "https://example.com/@user/pub/collections/abc123"
	externalContext := "https://other.example.com/context"

	legacyID := insertStreamWithContext(t, database, legacyURL, &legacyContext)
	validID := insertStreamWithContext(t, database, "https://example.com/stream/valid", &validContext)
	externalID := insertStreamWithContext(t, database, "https://example.com/stream/external", &externalContext)

	require.NoError(t, Version33(ctx, database))

	require.Equal(t, "", loadStreamContext(t, database, legacyID), "legacy context URLs must be cleared")
	require.Equal(t, validContext, loadStreamContext(t, database, validID), "valid collection URLs are left alone")
	require.Equal(t, externalContext, loadStreamContext(t, database, externalID), "external context URLs are left alone")
}

func TestVersion33_Idempotent(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	legacyURL := "https://example.com/stream/legacy"
	legacyContext := legacyURL + "/pub/context"

	legacyID := insertStreamWithContext(t, database, legacyURL, &legacyContext)

	require.NoError(t, Version33(ctx, database))
	require.NoError(t, Version33(ctx, database))

	require.Equal(t, "", loadStreamContext(t, database, legacyID), "a second run should make no additional changes")
}
