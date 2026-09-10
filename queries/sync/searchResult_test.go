package sync

import (
	"reflect"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/indexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// bsonFieldNames returns the bson name of every field on a struct type, following inlined embeds.
// It mirrors the helper of the same shape in the model package, which cannot be imported here.
func bsonFieldNames(structType reflect.Type) []string {

	result := make([]string, 0, structType.NumField())

	for index := range structType.NumField() {

		field := structType.Field(index)
		name, options, _ := strings.Cut(field.Tag.Get("bson"), ",")

		// An inlined struct (journal.Journal, etc) contributes its fields to the parent document
		if strings.Contains(options, "inline") {
			if field.Type.Kind() == reflect.Struct {
				result = append(result, bsonFieldNames(field.Type)...)
			}
			continue
		}

		if name != "" && name != "-" {
			result = append(result, name)
		}
	}

	return result
}

// indexKeys returns every field name used by any index in the set, deduplicated.
func indexKeys(indexes indexer.IndexSet) []string {

	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, index := range indexes {

		keys, ok := index.Keys.(bson.D)

		if !ok {
			continue
		}

		for _, key := range keys {
			if !seen[key.Key] {
				seen[key.Key] = true
				result = append(result, key.Key)
			}
		}
	}

	return result
}

// fieldsWrittenOutsideTheModel are real SearchResult fields that model.SearchResult does not
// declare, because the queries package writes them with raw Mongo updates that bypass the model
// (see queries/AGENTS.md). An index over one of these is legitimate.
var fieldsWrittenOutsideTheModel = []string{
	"lockId",      // queries.LockSearchResults
	"timeoutDate", // queries.LockSearchResults
}

// TestSearchResultIndexKeysExist guards every index key on the SearchResult collection
// against the model it indexes.
//
// An index key is a plain string that no compiler checks. A key naming a field that does not
// exist builds an EMPTY index, silently: MongoDB reports no error, the query planner never uses
// it, and the queries it was meant to serve quietly fall back to a collection scan. That is what
// `place.location` did here from 2025-10-30 until BUG-139 -- the field has always been `location`,
// and `places` was a *Stream* field that Version20 renamed years earlier.
func TestSearchResultIndexKeysExist(t *testing.T) {

	known := append(bsonFieldNames(reflect.TypeOf(model.SearchResult{})), fieldsWrittenOutsideTheModel...)

	// notifiedDate is indexed by idx_SearchResult_Notified and written by NOTHING. It looks
	// copied from the SearchQuery index set, where the field is real. Removing the index
	// changes how the search indexer's lock query plans, so it is filed rather than fixed here.
	known = append(known, "notifiedDate")

	for _, key := range indexKeys(searchResultIndexes()) {

		// Only the FIRST segment has to be a known field: a dotted key may reach into a
		// sub-document, but its root must exist or the index indexes nothing.
		root, _, _ := strings.Cut(key, ".")
		assert.Contains(t, known, root, "index key %q names no bson field on model.SearchResult", key)
	}
}

// TestSearchResultLocationIsGeospatiallyIndexable pins the pairing that makes the 2dsphere index
// work: the key must name the field, and the field must marshal as GeoJSON. `$geoIntersects`
// accepts GeoJSON only, so a bare coordinate array on either side is a silent miss.
func TestSearchResultLocationIsGeospatiallyIndexable(t *testing.T) {

	require.Contains(t, indexKeys(searchResultIndexes()), "location")

	fields := bsonFieldNames(reflect.TypeOf(model.SearchResult{}))
	require.Contains(t, fields, "location", "the 2dsphere index key must name a real field")

	// The shape half of the contract is pinned by model.TestSearchResult_LocationStoredAsGeoJSON
	require.NotContains(t, fields, "place", "`place` has never been a SearchResult field -- see BUG-139")
}
