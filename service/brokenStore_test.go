package service

import (
	"context"
	"reflect"

	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

/******************************************
 * Broken Store
 *
 * A data.Session whose collections list a fixed set
 * of records and fail every write, unless named as
 * writable.  The secrets tests use it to reach the
 * error paths that attach a record, without a live
 * database.
 ******************************************/

// brokenSession is a data.Session whose collections list fixed records and fail their writes
type brokenSession struct {
	records  map[string][]data.Object // The records each collection lists, by collection name
	writable map[string]bool          // Collections whose writes succeed
}

// Collection returns a brokenCollection over the named collection's records
func (session brokenSession) Collection(name string) data.Collection {
	return brokenCollection{records: session.records[name], writable: session.writable[name]}
}

// Context returns a background context
func (session brokenSession) Context() context.Context {
	return context.Background()
}

// Close does nothing
func (session brokenSession) Close() {}

// brokenCollection lists its records, finds nothing by criteria, and fails every write
// unless it is writable
type brokenCollection struct {
	data.Collection
	records  []data.Object
	writable bool
}

// Iterator lists every record, whatever the criteria
func (collection brokenCollection) Iterator(_ exp.Expression, _ ...option.Option) (data.Iterator, error) {
	return mockdb.NewIterator(collection.records), nil
}

// Query copies every record into target, which must point to a slice of the records' type
func (collection brokenCollection) Query(target any, _ exp.Expression, _ ...option.Option) error {

	slice := reflect.ValueOf(target).Elem()

	for _, record := range collection.records {
		slice.Set(reflect.Append(slice, reflect.ValueOf(record).Elem()))
	}

	return nil
}

// Load always fails with NotFound
func (collection brokenCollection) Load(_ exp.Expression, _ data.Object, _ ...option.Option) error {
	return derp.NotFound("service.brokenCollection.Load", "Synthetic failure")
}

// Save fails with an internal error, unless the collection is writable
func (collection brokenCollection) Save(_ data.Object, _ string) error {

	if collection.writable {
		return nil
	}

	return derp.Internal("service.brokenCollection.Save", "Synthetic failure")
}

// Delete fails with an internal error, unless the collection is writable
func (collection brokenCollection) Delete(_ data.Object, _ string) error {

	if collection.writable {
		return nil
	}

	return derp.Internal("service.brokenCollection.Delete", "Synthetic failure")
}
