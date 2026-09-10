package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests pin the KEY that removes a Stream or User from the search index. A SearchResult is a
// projection built fresh on every call, so it never carries the stored record's SearchResultID --
// only its URL is stable. Deleting by ID silently matches nothing, which is what left orphaned rows
// behind. data-mock cannot stand in here: its HardDelete is NotImplemented.

/******************************************
 * resultStore -- an in-memory data.Collection of SearchResults that matches
 * on the two fields the delete paths query: _id and url.
 ******************************************/

// resultStore is an in-memory data.Collection of SearchResults, used by the tests in this file
type resultStore struct {
	records []*model.SearchResult
}

// Context implements the interface, returning a background context
func (c *resultStore) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *resultStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.NotFound("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *resultStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *resultStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

// Load implements the data.Collection interface, backed by this stub's in-memory records
func (c *resultStore) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	for _, record := range c.records {
		if matchesResult(criteria, record) {
			if searchResult, ok := target.(*model.SearchResult); ok {
				*searchResult = *record
				return nil
			}
			return derp.Internal("test", "unexpected target type")
		}
	}

	return derp.NotFound("test", "not found")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *resultStore) Save(data.Object, string) error {
	return derp.NotFound("test", "unused")
}

// Delete implements the data.Collection interface. Unused by these tests, which delete hard.
func (c *resultStore) Delete(data.Object, string) error {
	return derp.NotFound("test", "unused")
}

// HardDelete implements the data.Collection interface, removing every matching record
func (c *resultStore) HardDelete(criteria exp.Expression) error {

	kept := make([]*model.SearchResult, 0, len(c.records))

	for _, record := range c.records {
		if !matchesResult(criteria, record) {
			kept = append(kept, record)
		}
	}

	c.records = kept
	return nil
}

// matchesResult reports whether a SearchResult satisfies the criteria, on the fields
// the delete paths actually query. Anything else conservatively counts as "no match".
func matchesResult(criteria exp.Expression, record *model.SearchResult) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		if predicate.Operator != exp.OperatorEqual {
			return false
		}

		switch predicate.Field {

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.SearchResultID == value

		case "url":
			value, ok := predicate.Value.(string)
			return ok && record.URL == value

		default:
			return false
		}
	})
}

// resultSession is a data.Session that hands out a single resultStore
type resultSession struct {
	store *resultStore
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s resultSession) Collection(string) data.Collection { return s.store }

// Context implements the interface, returning a background context
func (s resultSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s resultSession) Close() {}

// newResultService returns a SearchResult service backed by the provided store
func newResultService(store *resultStore) (*SearchResult, resultSession) {
	service := NewSearchResult()
	return &service, resultSession{store: store}
}

// newStoredResult returns a SearchResult as it would already exist in the database
func newStoredResult(url string) *model.SearchResult {
	result := model.NewSearchResult()
	result.URL = url
	result.Type = "Note"
	result.Name = "Test Placemark"
	return &result
}

/******************************************
 * Tests
 ******************************************/

// DeleteByURL removes the matching record and leaves the others intact.
func TestSearchResult_DeleteByURL(t *testing.T) {

	keep := newStoredResult("https://example.test/keep")
	remove := newStoredResult("https://example.test/remove")

	store := &resultStore{records: []*model.SearchResult{keep, remove}}
	service, session := newResultService(store)

	require.Nil(t, service.DeleteByURL(session, "https://example.test/remove"))

	require.Len(t, store.records, 1)
	require.Equal(t, "https://example.test/keep", store.records[0].URL)
}

// DeleteByURL is a no-op for a URL that was never indexed, and for an empty URL.
func TestSearchResult_DeleteByURL_Absent(t *testing.T) {

	existing := newStoredResult("https://example.test/keep")
	store := &resultStore{records: []*model.SearchResult{existing}}
	service, session := newResultService(store)

	require.Nil(t, service.DeleteByURL(session, "https://example.test/absent"))
	require.Nil(t, service.DeleteByURL(session, ""))
	require.Len(t, store.records, 1)
}

// Repeating DeleteByURL is safe, because the delete paths run it more than once:
// `unpublish` removes the row and the following `delete` asks again.
func TestSearchResult_DeleteByURL_Idempotent(t *testing.T) {

	store := &resultStore{records: []*model.SearchResult{newStoredResult("https://example.test/post")}}
	service, session := newResultService(store)

	require.Nil(t, service.DeleteByURL(session, "https://example.test/post"))
	require.Nil(t, service.DeleteByURL(session, "https://example.test/post"))
	require.Empty(t, store.records)
}

// A SearchResult built from a Stream carries a BRAND NEW SearchResultID, so deleting by
// that ID matches nothing. This is why every removal path keys on URL instead.
func TestSearchResult_DeleteByID_CannotMatchAProjection(t *testing.T) {

	const url = "https://example.test/post"

	stored := newStoredResult(url)
	store := &resultStore{records: []*model.SearchResult{stored}}
	service, session := newResultService(store)

	// A projection of the same object, as Stream.SearchResult would build it
	projection := model.NewSearchResult()
	projection.URL = url

	require.NotEqual(t, stored.SearchResultID, projection.SearchResultID)

	// Deleting by the projection's ID reports success while removing nothing
	require.Nil(t, service.Delete(session, &projection, "by id"))
	require.Len(t, store.records, 1)

	// Deleting by URL is what actually removes it
	require.Nil(t, service.DeleteByURL(session, url))
	require.Empty(t, store.records)
}
