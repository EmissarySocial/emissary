package upgrades

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestVersion37_BackfillsOnlyMissingTypes runs the migration twice against a real MongoDB,
// and confirms that records with no type become ACTOR while a known type is left alone.
func TestVersion37_BackfillsOnlyMissingTypes(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()
	collection := database.Collection("Annotation")

	missing := primitive.NewObjectID()
	empty := primitive.NewObjectID()
	note := primitive.NewObjectID()

	_, err := collection.InsertMany(ctx, []any{
		bson.M{"_id": missing, "url": "https://example.test/@alice"},
		bson.M{"_id": empty, "url": "https://example.test/@bob", "type": ""},
		bson.M{"_id": note, "url": "https://example.test/notes/1", "type": "DOCUMENT"},
	})
	require.NoError(t, err)

	// Running twice must be harmless
	require.NoError(t, Version37(ctx, database))
	require.NoError(t, Version37(ctx, database))

	typeOf := func(annotationID primitive.ObjectID) string {
		record := struct {
			Type string `bson:"type"`
		}{}
		require.NoError(t, collection.FindOne(ctx, bson.M{"_id": annotationID}).Decode(&record))
		return record.Type
	}

	require.Equal(t, "ACTOR", typeOf(missing))
	require.Equal(t, "ACTOR", typeOf(empty))
	require.Equal(t, "DOCUMENT", typeOf(note))
}
