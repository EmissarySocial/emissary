package model

import (
	"testing"

	"github.com/benpate/geo"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Stored format tests
 *
 * SearchResult.Location is a geo.Point, whose BSON shape changed
 * silently in geo v0.1.0 (BUG-139). These tests pin the shape this
 * package writes and prove that BOTH historical shapes still load.
 ******************************************/

// legacySearchResultID is fixed, so every document these tests build is byte-for-byte
// reproducible from one run to the next.
var legacySearchResultID = primitive.ObjectID{
	0x6a, 0xa0, 0x97, 0xb6, 0x31, 0x28, 0x0a, 0x03, 0x8d, 0xe0, 0x0d, 0x87,
}

// searchResultWithLocation builds one raw SearchResult document whose `location`
// carries the supplied value, exactly as a row on disk would look.
func searchResultWithLocation(t *testing.T, value any) []byte {

	t.Helper()

	document := bson.M{
		"_id":  legacySearchResultID,
		"type": "Event",
		"url":  "https://bandwagon.fm/@venue/1234",
		"name": "Live at the Bluebird",
	}

	if value != nil {
		document["location"] = value
	}

	data, err := bson.Marshal(document)
	require.Nil(t, err)

	return data
}

// TestSearchResult_LocationStoredAsGeoJSON pins the shape this package WRITES.
// A bare `[lon,lat]` array here means geo.Point has regressed again.
func TestSearchResult_LocationStoredAsGeoJSON(t *testing.T) {

	searchResult := NewSearchResult()
	searchResult.Location = geo.NewPoint(-104.9903, 39.7392)

	data, err := bson.Marshal(searchResult)
	require.Nil(t, err)

	location := bson.Raw(data).Lookup("location")
	require.Equal(t, bson.TypeEmbeddedDocument, location.Type, "location must be a GeoJSON document, not a bare array")
	require.Equal(t, "Point", location.Document().Lookup("type").StringValue())
}

// TestSearchResult_ZeroLocationIsOmitted pins the `omitempty` behavior that keeps
// ungeocoded results out of the 2dsphere index entirely.
func TestSearchResult_ZeroLocationIsOmitted(t *testing.T) {

	data, err := bson.Marshal(NewSearchResult())
	require.Nil(t, err)

	_, err = bson.Raw(data).LookupErr("location")
	require.NotNil(t, err, "a zero location must not be stored at all")
}

// TestSearchResult_LoadsEveryHistoricalLocationShape is the regression test for the
// production 500 on bandwagon.fm/events. Both shapes are live in the collection, so
// both must decode -- before ANY migration runs.
func TestSearchResult_LoadsEveryHistoricalLocationShape(t *testing.T) {

	// shapes are the on-disk forms this field has had, newest last
	shapes := map[string]any{
		"GeoJSONDocument_beforeGeoV010": bson.M{"type": "Point", "coordinates": bson.A{-104.9903, 39.7392}},
		"BareArray_geoV010ThroughV020":  bson.A{-104.9903, 39.7392},
		"BareArrayWithAltitude":         bson.A{-104.9903, 39.7392, 1609.0},
		"GeoJSONDocumentWithAltitude":   bson.M{"type": "Point", "coordinates": bson.A{-104.9903, 39.7392, 1609.0}},
	}

	for name, shape := range shapes {

		t.Run(name, func(t *testing.T) {

			searchResult := NewSearchResult()
			require.NoError(t, bson.Unmarshal(searchResultWithLocation(t, shape), &searchResult))

			require.InDelta(t, -104.9903, searchResult.Location.Longitude, 0.00001)
			require.InDelta(t, 39.7392, searchResult.Location.Latitude, 0.00001)
			require.Equal(t, "Live at the Bluebird", searchResult.Name)
		})
	}
}

// TestSearchResult_LoadsSliceContainingEveryShape reproduces the exact production
// failure: one undecodable record in a cursor takes the WHOLE page down, because
// data-mongo decodes a query with cursor.All.
func TestSearchResult_LoadsSliceContainingEveryShape(t *testing.T) {

	documents := [][]byte{
		searchResultWithLocation(t, bson.M{"type": "Point", "coordinates": bson.A{-104.9903, 39.7392}}),
		searchResultWithLocation(t, bson.A{-104.9903, 39.7392}),
		searchResultWithLocation(t, nil),
	}

	for index, document := range documents {
		searchResult := NewSearchResult()
		require.NoError(t, bson.Unmarshal(document, &searchResult), "record %d failed to decode", index)
	}
}

// TestSearchResult_MissingAndNullLocation confirms an absent or NULL location is the
// zero Point rather than an error. Most SearchResults have no location at all.
func TestSearchResult_MissingAndNullLocation(t *testing.T) {

	missing := NewSearchResult()
	require.NoError(t, bson.Unmarshal(searchResultWithLocation(t, nil), &missing))
	require.True(t, missing.Location.IsZero())

	null := NewSearchResult()
	require.NoError(t, bson.Unmarshal(searchResultWithLocation(t, primitive.Null{}), &null))
	require.True(t, null.Location.IsZero())
}

// TestSearchResult_LocationRoundTripIsByteStable asserts bytes -> struct -> bytes is
// identity for the current shape, which is what makes the Version30 migration a
// safe no-op on records that are already correct.
func TestSearchResult_LocationRoundTripIsByteStable(t *testing.T) {

	searchResult := NewSearchResult()
	searchResult.Location = geo.NewPointWithAltitude(-104.9903, 39.7392, 1609)

	once, err := bson.Marshal(searchResult)
	require.Nil(t, err)

	reloaded := NewSearchResult()
	require.NoError(t, bson.Unmarshal(once, &reloaded))

	twice, err := bson.Marshal(reloaded)
	require.Nil(t, err)

	require.Equal(t, once, twice)
}
