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

// These drive Version30 against a REAL MongoDB, because the behavior that matters belongs to the
// server: whether an aggregation-pipeline `$set` can wrap a field in a document that references
// itself, and whether the `$type: "array"` filter distinguishes the two on-disk shapes. They skip
// when no database is reachable, so `go test ./...` still passes without one.

// insertSearchResultLocation writes one raw SearchResult document. Pass nil to write a
// record with NO location key, as an ungeocoded result would look.
func insertSearchResultLocation(t *testing.T, database *mongo.Database, value any) primitive.ObjectID {

	t.Helper()

	id := primitive.NewObjectID()
	document := bson.M{"_id": id, "url": "https://bandwagon.fm/@venue/" + id.Hex(), "name": "Test Event"}

	if value != nil {
		document["location"] = value
	}

	_, err := database.Collection("SearchResult").InsertOne(context.Background(), document)
	require.NoError(t, err)

	return id
}

// rawLocation reads back the `location` field of one SearchResult, exactly as stored.
func rawLocation(t *testing.T, database *mongo.Database, id primitive.ObjectID) bson.RawValue {

	t.Helper()

	raw := bson.Raw{}
	err := database.Collection("SearchResult").FindOne(context.Background(), bson.M{"_id": id}).Decode(&raw)
	require.NoError(t, err)

	return raw.Lookup("location")
}

// TestVersion30_NormalizesLegacyArrays is the core of the migration: a bare coordinate
// array becomes the GeoJSON document that `$geoIntersects` and the 2dsphere index need.
func TestVersion30_NormalizesLegacyArrays(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	pair := insertSearchResultLocation(t, database, bson.A{-104.9903, 39.7392})
	triple := insertSearchResultLocation(t, database, bson.A{-104.9903, 39.7392, 1609.0})

	require.NoError(t, Version30(ctx, database))

	for _, id := range []primitive.ObjectID{pair, triple} {
		location := rawLocation(t, database, id)
		require.Equal(t, bson.TypeEmbeddedDocument, location.Type)
		require.Equal(t, "Point", location.Document().Lookup("type").StringValue())
	}

	// The coordinates must survive the rewrite untouched
	searchResult := model.NewSearchResult()
	require.NoError(t, database.Collection("SearchResult").FindOne(ctx, bson.M{"_id": triple}).Decode(&searchResult))
	require.InDelta(t, -104.9903, searchResult.Location.Longitude, 0.00001)
	require.InDelta(t, 39.7392, searchResult.Location.Latitude, 0.00001)
	require.InDelta(t, 1609.0, searchResult.Location.Altitude, 0.00001)
}

// TestVersion30_LeavesCurrentShapeAlone confirms the filter does not touch a record
// that is already correct, which is what keeps the migration cheap and safe.
func TestVersion30_LeavesCurrentShapeAlone(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	id := insertSearchResultLocation(t, database, bson.M{"type": "Point", "coordinates": bson.A{-104.9903, 39.7392}})
	before := rawLocation(t, database, id)

	require.NoError(t, Version30(ctx, database))

	require.Equal(t, before, rawLocation(t, database, id))
}

// TestVersion30_IsIdempotent runs the migration twice. The second pass must find
// nothing, because nothing it writes is an array.
func TestVersion30_IsIdempotent(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	id := insertSearchResultLocation(t, database, bson.A{-104.9903, 39.7392})

	require.NoError(t, Version30(ctx, database))
	once := rawLocation(t, database, id)

	require.NoError(t, Version30(ctx, database))
	twice := rawLocation(t, database, id)

	require.Equal(t, once, twice)
}

// TestVersion30_SkipsMalformedArrays confirms the length guards. A coordinate array
// that is not [lon,lat] or [lon,lat,alt] is left for a human rather than guessed at.
func TestVersion30_SkipsMalformedArrays(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	empty := insertSearchResultLocation(t, database, bson.A{})
	single := insertSearchResultLocation(t, database, bson.A{-104.9903})
	tooMany := insertSearchResultLocation(t, database, bson.A{1.0, 2.0, 3.0, 4.0})

	require.NoError(t, Version30(ctx, database))

	for _, id := range []primitive.ObjectID{empty, single, tooMany} {
		require.Equal(t, bson.TypeArray, rawLocation(t, database, id).Type, "malformed arrays must be left alone")
	}
}

// TestVersion30_SkipsNonNumericCoordinates confirms a coordinate array holding something
// that is not a number is reported and left alone, rather than aborting the migration.
func TestVersion30_SkipsNonNumericCoordinates(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	corrupt := insertSearchResultLocation(t, database, bson.A{"west", "north"})
	healthy := insertSearchResultLocation(t, database, bson.A{-104.9903, 39.7392})

	require.NoError(t, Version30(ctx, database))

	// The corrupt record is untouched...
	require.Equal(t, bson.TypeArray, rawLocation(t, database, corrupt).Type)

	// ...and it did not stop the healthy one from being repaired
	require.Equal(t, bson.TypeEmbeddedDocument, rawLocation(t, database, healthy).Type)
}

// TestVersion30_LeavesUngeocodedRecordsAlone confirms a record with no location at all
// is untouched. Most SearchResults are in this state, so it is the common case.
func TestVersion30_LeavesUngeocodedRecordsAlone(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	id := insertSearchResultLocation(t, database, nil)

	require.NoError(t, Version30(ctx, database))

	raw := bson.Raw{}
	require.NoError(t, database.Collection("SearchResult").FindOne(ctx, bson.M{"_id": id}).Decode(&raw))

	_, err := raw.LookupErr("location")
	require.NotNil(t, err, "an absent location must stay absent")
}

// TestVersion30_MixedCollectionDecodes reproduces the production failure end to end:
// a cursor holding BOTH shapes must decode, because data-mongo reads a query with
// cursor.All and one bad record fails the whole page.
func TestVersion30_MixedCollectionDecodes(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	ctx := context.Background()

	insertSearchResultLocation(t, database, bson.M{"type": "Point", "coordinates": bson.A{-104.9903, 39.7392}})
	insertSearchResultLocation(t, database, bson.A{-104.9903, 39.7392})
	insertSearchResultLocation(t, database, nil)

	// The mixed collection must decode BEFORE the migration runs -- that is the library fix
	cursor, err := database.Collection("SearchResult").Find(ctx, bson.M{})
	require.NoError(t, err)

	before := make([]model.SearchResult, 0)
	require.NoError(t, cursor.All(ctx, &before))
	require.Equal(t, 3, len(before))

	// ...and again afterwards, now that every location is one shape
	require.NoError(t, Version30(ctx, database))

	cursor, err = database.Collection("SearchResult").Find(ctx, bson.M{})
	require.NoError(t, err)

	after := make([]model.SearchResult, 0)
	require.NoError(t, cursor.All(ctx, &after))
	require.Equal(t, 3, len(after))
}
