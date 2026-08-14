package ascache

import (
	"context"
	"slices"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

// The stock data-mock server cannot back these tests: its HardDelete is unimplemented (which aborts
// every cache write inside removeDuplicates), and its reflection matcher compares a `urls` SLICE
// against a single URL, so a saved value could never be found again. Both are exactly the operations
// the cache read/write path is built on, so the fake below implements those two -- and only those two.

// fakeServer is an in-memory data.Server holding cached Values for one test.
type fakeServer struct {
	values []Value // Every value saved so far, in insertion order
}

// newFakeServer returns an empty in-memory database.
func newFakeServer() *fakeServer {
	return &fakeServer{
		values: make([]Value, 0),
	}
}

// Session returns a session bound to this server.
func (server *fakeServer) Session(ctx context.Context) (data.Session, error) {
	return &fakeSession{server: server, ctx: ctx}, nil
}

// WithTransaction runs the callback against a session.  There is no rollback: a test that needs one
// is testing the database, not the cache.
func (server *fakeServer) WithTransaction(ctx context.Context, fn data.TransactionCallbackFunc) (any, error) {

	session, err := server.Session(ctx)

	if err != nil {
		return nil, err
	}

	return fn(session)
}

// fakeSession is a data.Session over a fakeServer.
type fakeSession struct {
	server *fakeServer     // Underlying storage
	ctx    context.Context // Context this session was opened with
}

// Collection returns the named collection.  All names share one store, because the cache uses one.
func (session *fakeSession) Collection(name string) data.Collection {
	return &fakeCollection{server: session.server}
}

// Context returns this session's context.
func (session *fakeSession) Context() context.Context {
	return session.ctx
}

// Close satisfies data.Session.  Nothing to release.
func (session *fakeSession) Close() {}

// fakeCollection implements the slice of data.Collection that ascache.Client actually calls.
type fakeCollection struct {
	server *fakeServer // Underlying storage
}

// Context satisfies data.Collection.  Nothing here is context-aware.
func (collection *fakeCollection) Context() context.Context {
	return context.Background()
}

// Load returns the first stored Value matching the criteria.
func (collection *fakeCollection) Load(criteria exp.Expression, target data.Object, options ...option.Option) error {

	for _, value := range collection.server.values {

		if criteria.Match(matchURLs(value)) {

			if typed, ok := target.(*Value); ok {
				*typed = value
			}

			return nil
		}
	}

	return derp.NotFound("ascache.fakeCollection.Load", "Document not found", criteria)
}

// Save inserts or replaces a Value, keyed by its ValueID.
func (collection *fakeCollection) Save(object data.Object, note string) error {

	typed, ok := object.(*Value)

	if !ok {
		return derp.Internal("ascache.fakeCollection.Save", "Object must be a *Value", object)
	}

	for index, existing := range collection.server.values {
		if existing.ValueID == typed.ValueID {
			collection.server.values[index] = *typed
			return nil
		}
	}

	collection.server.values = append(collection.server.values, *typed)
	return nil
}

// HardDelete removes every stored Value matching the criteria.
func (collection *fakeCollection) HardDelete(criteria exp.Expression) error {

	collection.server.values = slices.DeleteFunc(collection.server.values, func(value Value) bool {
		return criteria.Match(matchURLs(value))
	})

	return nil
}

// Delete removes a Value outright.  The cache's soft-delete path is not under test.
func (collection *fakeCollection) Delete(object data.Object, note string) error {
	return derp.NotImplemented("ascache.fakeCollection.Delete", "Not needed by these tests")
}

// Count satisfies data.Collection.  Unused by the cache read/write path.
func (collection *fakeCollection) Count(criteria exp.Expression, options ...option.Option) (int64, error) {
	return 0, derp.NotImplemented("ascache.fakeCollection.Count", "Not needed by these tests")
}

// Query satisfies data.Collection.  Unused by the cache read/write path.
func (collection *fakeCollection) Query(target any, criteria exp.Expression, options ...option.Option) error {
	return derp.NotImplemented("ascache.fakeCollection.Query", "Not needed by these tests")
}

// Iterator satisfies data.Collection.  Unused by the cache read/write path.
func (collection *fakeCollection) Iterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return nil, derp.NotImplemented("ascache.fakeCollection.Iterator", "Not needed by these tests")
}

// matchURLs returns a matcher for the two `urls` predicates the cache issues: an EQUAL from
// loadByURL, and an IN from removeDuplicates.  Both mean "this value answers to that URL".
func matchURLs(value Value) exp.MatcherFunc {

	return func(predicate exp.Predicate) bool {

		if predicate.Field != "urls" {
			return false
		}

		switch predicate.Operator {

		case exp.OperatorEqual:
			url, ok := predicate.Value.(string)
			return ok && slices.Contains(value.URLs, url)

		case exp.OperatorIn:
			urls, ok := predicate.Value.([]string)

			if !ok {
				return false
			}

			return slices.ContainsFunc(urls, func(url string) bool {
				return slices.Contains(value.URLs, url)
			})
		}

		return false
	}
}
