package upgrades

import (
	"context"
	"fmt"

	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Version30 rewrites every SearchResult.location that is a bare coordinate array
// back into the GeoJSON document that the rest of the system expects.
//
// geo v0.1.0 moved Position onto bson.ValueMarshaler while Point kept bson.Marshaler. The two
// interfaces use DIFFERENT method names, so nothing shadowed anything: the driver preferred the
// pair promoted from the embedded Position, and Point silently began writing `[lon,lat]` instead
// of `{"type":"Point","coordinates":[...]}`. Emissary took that version on 2026-06-28, so the
// collection now holds both shapes. See BUG-139.
//
// geo's decoder accepts both shapes permanently, so this is NOT what makes those records readable
// again -- the library fix is. This exists because `$geoIntersects` requires GeoJSON, because the
// 2dsphere index should cover one shape rather than two, and because a field with two shapes
// cannot be reasoned about.
func Version30(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.Version30"

	fmt.Println("... Version 30")

	if err := normalizeSearchResultLocations(ctx, session); err != nil {
		return derp.Wrap(err, location, "Normalizing SearchResult locations")
	}

	return nil
}

// legacyLocation is one SearchResult carrying the bare coordinate array that geo v0.1.0 wrote.
type legacyLocation struct {
	ID       any       `bson:"_id"`
	Location []float64 `bson:"location"`
}

// normalizeSearchResultLocations converts array-shaped `location` values into GeoJSON documents.
//
// It is idempotent by construction: the filter selects only arrays, and nothing it writes is an
// array, so a second run matches nothing. Records already in GeoJSON form are never touched.
func normalizeSearchResultLocations(ctx context.Context, session *mongo.Database) error {

	const location = "queries.upgrades.normalizeSearchResultLocations"

	collection := session.Collection("SearchResult")

	// Find every record whose location is still a bare coordinate array
	cursor, err := collection.Find(ctx, bson.M{"location": bson.M{"$type": "array"}})

	if err != nil {
		return derp.Wrap(err, location, "Listing SearchResults with array coordinates")
	}

	// Plan one update per record.
	//
	// This is deliberately NOT an aggregation-pipeline update. In pipeline context, `$set` on a
	// field that currently holds an array broadcasts the new document across the array's elements
	// instead of replacing it, which quietly produces an array of identical Points.
	writes := make([]mongo.WriteModel, 0)
	malformed := 0

	for cursor.Next(ctx) {

		record := legacyLocation{}

		// A coordinate that is not a number cannot be repaired here, so report it and move on
		if err := cursor.Decode(&record); err != nil {
			derp.Report(derp.Wrap(err, location, "Decoding SearchResult location"))
			malformed++
			continue
		}

		// RULE: A GeoJSON position is [lon, lat] or [lon, lat, alt]. Anything else is
		// left for a human rather than guessed at.
		if len(record.Location) < 2 || len(record.Location) > 3 {
			malformed++
			continue
		}

		geoJSON := bson.M{"type": "Point", "coordinates": record.Location}

		writes = append(writes, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": record.ID}).
			SetUpdate(bson.M{"$set": bson.M{"location": geoJSON}}))
	}

	if err := cursor.Err(); err != nil {
		return derp.Wrap(err, location, "Reading SearchResults from cursor")
	}

	// An install with no legacy coordinates is the common case, and BulkWrite rejects an empty batch
	if len(writes) == 0 {
		fmt.Println("...... no SearchResult locations needed normalizing")
		return reportMalformedLocations(malformed)
	}

	// Apply every rewrite in a single round-trip
	result, err := collection.BulkWrite(ctx, writes, options.BulkWrite().SetOrdered(false))

	if err != nil {
		return derp.Wrap(err, location, "Rewriting coordinate arrays as GeoJSON")
	}

	// The driver pairs a nil result with a non-nil error on every path it takes today, but that is
	// an internal detail rather than a documented promise, and the count is only used for the log.
	if result == nil {
		fmt.Println("...... normalized an unreported number of SearchResult locations")
	} else {
		fmt.Println("...... normalized " + fmt.Sprint(result.ModifiedCount) + " SearchResult locations")
	}

	return reportMalformedLocations(malformed)
}

// reportMalformedLocations warns about coordinate arrays this migration refused to touch.
func reportMalformedLocations(malformed int) error {

	if malformed > 0 {
		fmt.Println("...... WARNING: " + fmt.Sprint(malformed) + " SearchResult locations are malformed arrays and were left alone")
	}

	// Location, location, location.
	return nil
}
