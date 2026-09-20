package model

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/tools/datetime"
	"github.com/benpate/delta"
	"github.com/benpate/geo"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * BSON wire-format tests
 *
 * These pin the Extended JSON Emissary writes for the field types
 * that carry a CUSTOM BSON marshaller, with every document sorted
 * by key. Everything else rides the driver's default codecs.
 * See AGENTS.md.
 ******************************************/

// Every type Emissary persists through a custom BSON marshaller must keep satisfying
// the interface it relies on. Losing one is not a compile error at the type's own
// definition: the driver silently falls back to the default struct codec instead.
var (
	_ bson.ValueMarshaler   = datetime.DateTime{}
	_ bson.ValueUnmarshaler = &datetime.DateTime{}

	_ bson.ValueMarshaler   = geo.Point{}
	_ bson.ValueUnmarshaler = &geo.Point{}
	_ bson.Marshaler        = geo.Polygon{}
	_ bson.Unmarshaler      = &geo.Polygon{}

	_ bson.ValueMarshaler   = delta.Bool{}
	_ bson.ValueUnmarshaler = &delta.Bool{}
	_ bson.ValueMarshaler   = delta.ObjectID{}
	_ bson.ValueUnmarshaler = &delta.ObjectID{}
)

// updateGolden rewrites the checked-in fixture instead of asserting against it.
// Run `go test ./model/ -update-golden` after an INTENDED format change, and read the diff.
var updateGolden = flag.Bool("update-golden", false, "rewrite the testdata/golden fixtures")

// bsonWireRecord stands in for a stored document that carries every
// custom-marshalled field type at once, so one fixture pins them all.
type bsonWireRecord struct {
	StartDate datetime.DateTime `bson:"startDate"`
	Location  geo.Point         `bson:"location"`
	Area      geo.Polygon       `bson:"area"`
	Where     geo.Address       `bson:"where"`
	IsBridged delta.Bool        `bson:"isBridged"`
	FolderID  delta.ObjectID    `bson:"folderId"`
	Data      mapof.Any         `bson:"data"`
}

// fullBSONWireRecord returns a record whose every field carries a distinctive
// non-zero value, because `omitempty` hides a zero field from these assertions.
func fullBSONWireRecord(t *testing.T) bsonWireRecord {

	t.Helper()

	folderID, err := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
	require.Nil(t, err)

	return bsonWireRecord{
		StartDate: datetime.DateTime{Time: time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)},
		Location:  geo.NewPointWithAltitude(-104.9903, 39.7392, 1609),
		Area:      geo.NewPolygon(geo.NewPosition(-105, 39), geo.NewPosition(-104, 39), geo.NewPosition(-104, 40)),
		Where: geo.Address{
			Name:       "Bluebird Theater",
			Formatted:  "3317 E Colfax Ave, Denver, CO 80206",
			Street1:    "3317 E Colfax Ave",
			Street2:    "Suite 2",
			Locality:   "Denver",
			Region:     "CO",
			PostalCode: "80206",
			Country:    "US",
			PlusCode:   "85FPQXX5+9F",
			Timezone:   "America/Denver",
			Latitude:   39.7392,
			Longitude:  -104.9903,
		},
		IsBridged: delta.NewBool(true),
		FolderID:  delta.NewObjectID(folderID),
		Data: mapof.Any{
			"scalar": "hello",
			"number": int64(42),
			"nested": mapof.Any{"inner": "value"},
		},
	}
}

// extendedJSON renders raw BSON as indented canonical Extended JSON, so a format
// change lands in a review diff as readable lines instead of a wall of bytes.
func extendedJSON(t *testing.T, data []byte) string {

	t.Helper()

	// RULE: Every document is re-keyed in sorted order, because the driver writes a Go
	// map in a different order on every run, and a fixture cannot pin one of them
	decoder := json.NewDecoder(strings.NewReader(bson.Raw(data).String()))
	decoder.UseNumber() // A number decoded as a float64 would be re-rendered, losing digits

	document := map[string]any{}
	require.NoError(t, decoder.Decode(&document))

	// Render it back with the values exactly as the driver wrote them
	buffer := bytes.Buffer{}
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	require.NoError(t, encoder.Encode(document)) // Encode ends the document with a newline

	return buffer.String()
}

// requireGoldenBSON asserts that `actual` matches the named fixture, or rewrites
// it under -update-golden.
func requireGoldenBSON(t *testing.T, name string, actual string) {

	t.Helper()

	path := filepath.Join("testdata", "golden", name)

	// Rewrite the fixture when the format change is intended
	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(actual), 0600))
		return
	}

	// Otherwise compare against what is checked in
	expected, err := os.ReadFile(path)
	require.NoError(t, err, "missing fixture %s -- run `go test ./model/ -update-golden`", name)
	require.Equal(t, string(expected), actual,
		"format changed. If this is intended, run `go test ./model/ -update-golden` and review the diff")
}

// TestBSONWireFormat pins the Extended JSON written for every custom-marshalled
// field type Emissary persists.
func TestBSONWireFormat(t *testing.T) {

	record := fullBSONWireRecord(t)

	// Marshalled as FIELDS of a record, never each at the top level: a top-level
	// Marshal takes the Marshaler path directly and never consults the codec
	// registry, so it cannot observe a type losing its marshaller.
	data, err := bson.Marshal(record)
	require.Nil(t, err)

	requireGoldenBSON(t, "wireFormat.json", extendedJSON(t, data))
}

// TestBSONWireFormat_RenderIsStable proves the fixture cannot fail on a run that
// changed nothing.  `data` is a Go map, which the driver writes in a different order
// every time, so an unsorted rendering matches twenty times over by a chance of
// roughly one in three billion.
func TestBSONWireFormat_RenderIsStable(t *testing.T) {

	const attempts = 20

	data, err := bson.Marshal(fullBSONWireRecord(t))
	require.Nil(t, err)

	first := extendedJSON(t, data)

	for attempt := range attempts {

		// Marshalled again each time, because the order is chosen while the map is read
		data, err := bson.Marshal(fullBSONWireRecord(t))
		require.Nil(t, err)

		require.Equal(t, first, extendedJSON(t, data), "rendering moved on attempt %d", attempt)
	}
}

// TestBSONWireFormat_RoundTrip confirms the record decodes back to the value it
// was stored from, so the fixture describes a shape that is actually readable.
func TestBSONWireFormat_RoundTrip(t *testing.T) {

	original := fullBSONWireRecord(t)

	data, err := bson.Marshal(original)
	require.Nil(t, err)

	result := bsonWireRecord{}
	require.Nil(t, bson.Unmarshal(data, &result))

	require.True(t, original.StartDate.Equal(result.StartDate.Time), "startDate")
	require.Equal(t, original.Location, result.Location, "location")
	require.Equal(t, original.Area, result.Area, "area")
	require.Equal(t, original.Where, result.Where, "where")
	require.Equal(t, original.IsBridged.Value(), result.IsBridged.Value(), "isBridged")
	require.Equal(t, original.FolderID.Value(), result.FolderID.Value(), "folderId")
}

// TestBSONWireFormat_NestedMapAccessors pins the decoded TYPE of a document
// nested inside a mapof.Any, which is what the rosetta accessors match on.
func TestBSONWireFormat_NestedMapAccessors(t *testing.T) {

	data, err := bson.Marshal(fullBSONWireRecord(t))
	require.Nil(t, err)

	result := bsonWireRecord{}
	require.Nil(t, bson.Unmarshal(data, &result))

	// RULE: a nested document must stay reachable through the accessors. A driver
	// that decodes it as an ordered bson.D fails these assertions silently
	// returning empty, rather than reporting an error.
	require.Equal(t, "value", result.Data.GetMap("nested").GetString("inner"))
	require.Equal(t, "value", result.Data.GetMapOfAny("nested").GetString("inner"))
	require.Equal(t, "hello", result.Data.GetString("scalar"))
	require.Equal(t, int64(42), result.Data.GetInt64("number"))
}
