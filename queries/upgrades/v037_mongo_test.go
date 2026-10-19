package upgrades

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestVersion37_NormalizesEveryType runs the migration twice against a real MongoDB, starting from
// a missing type, an empty one, raw ActivityStreams types, and the two values it writes.
func TestVersion37_NormalizesEveryType(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()
	collection := database.Collection("Annotation")

	expected := map[primitive.ObjectID]string{}
	insert := func(document bson.M, want string) {
		annotationID := primitive.NewObjectID()
		document["_id"] = annotationID
		_, err := collection.InsertOne(ctx, document)
		require.NoError(t, err)
		expected[annotationID] = want
	}

	insert(bson.M{}, "ACTOR")
	insert(bson.M{"type": ""}, "ACTOR")
	insert(bson.M{"type": "Person"}, "ACTOR")
	insert(bson.M{"type": "Group"}, "ACTOR")
	insert(bson.M{"type": "ACTOR"}, "ACTOR")
	insert(bson.M{"type": "Note"}, "DOCUMENT")
	insert(bson.M{"type": "Article"}, "DOCUMENT")
	insert(bson.M{"type": "DOCUMENT"}, "DOCUMENT")

	// Running twice must be harmless
	require.NoError(t, Version37(ctx, database))
	require.NoError(t, Version37(ctx, database))

	for annotationID, want := range expected {
		record := struct {
			Type string `bson:"type"`
		}{}
		require.NoError(t, collection.FindOne(ctx, bson.M{"_id": annotationID}).Decode(&record))
		require.Equal(t, want, record.Type)
	}
}
