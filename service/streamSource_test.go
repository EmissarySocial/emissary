package service

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests use a hand-built data.Collection fake, for the reason given in follower_test.go:
// benpate/data-mock cannot see `deleteDate` inside the inlined journal.Journal.

/******************************************
 * In-Memory Fakes
 ******************************************/

// streamSourceCollection is an in-memory data.Collection that holds model.StreamSource records
type streamSourceCollection struct {
	records []model.StreamSource
	saved   []model.StreamSource // every record passed to Save, in order
}

// Context implements the data.Collection interface, returning a background context
func (c *streamSourceCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface
func (c *streamSourceCollection) Count(criteria exp.Expression, _ ...option.Option) (int64, error) {

	var count int64

	for _, record := range c.records {
		if matchesStreamSource(criteria, record) {
			count++
		}
	}

	return count, nil
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *streamSourceCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface
func (c *streamSourceCollection) Iterator(criteria exp.Expression, _ ...option.Option) (data.Iterator, error) {

	result := make([]model.StreamSource, 0)

	for _, record := range c.records {
		if matchesStreamSource(criteria, record) {
			result = append(result, record)
		}
	}

	return &streamSourceIterator{records: result}, nil
}

// Load copies the first matching StreamSource record into the target
func (c *streamSourceCollection) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	for _, record := range c.records {

		if !matchesStreamSource(criteria, record) {
			continue
		}

		streamSource, ok := target.(*model.StreamSource)

		if !ok {
			return derp.Internal("test", "unexpected target type")
		}

		*streamSource = record
		return nil
	}

	return derp.NotFound("test", "not found")
}

// Save upserts a StreamSource record, and remembers that it was asked to
func (c *streamSourceCollection) Save(object data.Object, _ string) error {

	streamSource, ok := object.(*model.StreamSource)

	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	c.saved = append(c.saved, *streamSource)

	for index, record := range c.records {
		if record.StreamSourceID == streamSource.StreamSourceID {
			c.records[index] = *streamSource
			return nil
		}
	}

	c.records = append(c.records, *streamSource)
	return nil
}

// Delete marks a StreamSource record deleted
func (c *streamSourceCollection) Delete(object data.Object, _ string) error {

	streamSource, ok := object.(*model.StreamSource)

	if !ok {
		return derp.Internal("test", "unexpected object type")
	}

	for index, record := range c.records {
		if record.StreamSourceID == streamSource.StreamSourceID {
			c.records[index].DeleteDate = 1
		}
	}

	return nil
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *streamSourceCollection) HardDelete(exp.Expression) error {
	return derp.Internal("test", "unused")
}

// streamSourceIterator walks a fixed slice of StreamSource records. Implements data.Iterator.
type streamSourceIterator struct {
	records []model.StreamSource
	index   int
}

// Next copies the next record into the target, returning FALSE when the list is exhausted
func (i *streamSourceIterator) Next(target any) bool {

	if i.index >= len(i.records) {
		return false
	}

	streamSource, ok := target.(*model.StreamSource)

	if !ok {
		return false
	}

	*streamSource = i.records[i.index]
	i.index++

	return true
}

// Count returns the number of records this iterator walks
func (i *streamSourceIterator) Count() int { return len(i.records) }

// Close releases this iterator. It holds nothing.
func (i *streamSourceIterator) Close() error { return nil }

// Error returns the error encountered while iterating, of which there are none
func (i *streamSourceIterator) Error() error { return nil }

// matchesStreamSource reports whether a record satisfies a criteria built from the predicates that
// the StreamSource service uses
func matchesStreamSource(criteria exp.Expression, record model.StreamSource) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		switch predicate.Field {

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && (predicate.Operator == exp.OperatorEqual) && (record.StreamSourceID == value)

		case "streamId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && (predicate.Operator == exp.OperatorEqual) && (record.StreamID == value)

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && (predicate.Operator == exp.OperatorEqual) && (record.DeleteDate == int64(value))
		}

		// RULE: An unsupported field is never a match
		return false
	})
}

// streamSourceSession hands out a single shared streamSourceCollection
type streamSourceSession struct {
	collection *streamSourceCollection
}

// Collection implements the data.Session interface
func (s streamSourceSession) Collection(string) data.Collection { return s.collection }

// Context implements the data.Session interface
func (s streamSourceSession) Context() context.Context { return context.Background() }

// Close implements the data.Session interface. The stub holds no resources to release.
func (s streamSourceSession) Close() {}

// newStreamSourceService returns a StreamSource service backed by an in-memory set of records
func newStreamSourceService(records ...model.StreamSource) (*StreamSource, streamSourceSession) {
	service := NewStreamSource()
	return &service, streamSourceSession{collection: &streamSourceCollection{records: records}}
}

// validStreamSource returns a StreamSource record that passes validation
func validStreamSource() model.StreamSource {

	result := model.NewStreamSource()
	result.StreamID = primitive.NewObjectID()
	result.Method = model.StreamSourceMethodHTTPS
	result.URL = "https://github.com/EmissarySocial/docs"
	result.Config["path"] = "docs/start.md"

	return result
}

/******************************************
 * Saving and Loading
 ******************************************/

// TestStreamSource_Save stores a valid record
func TestStreamSource_Save(t *testing.T) {

	service, session := newStreamSourceService()
	streamSource := validStreamSource()

	require.NoError(t, service.Save(session, &streamSource, "Created"))
	require.Len(t, session.collection.saved, 1)
	require.Equal(t, streamSource.StreamSourceID, session.collection.saved[0].StreamSourceID)
}

// TestStreamSource_Save_Refuses writes nothing for a record that is not valid, and reports every
// refusal as a client error, so an author's mistake never reaches the error log
func TestStreamSource_Save_Refuses(t *testing.T) {

	// refuses asserts that a broken record is refused with a status code in the provided range, and
	// never stored
	refuses := func(name string, lowest int, highest int, breakRecord func(*model.StreamSource)) {
		t.Run(name, func(t *testing.T) {

			service, session := newStreamSourceService()
			streamSource := validStreamSource()
			breakRecord(&streamSource)

			err := service.Save(session, &streamSource, "Created")

			require.Error(t, err)
			require.GreaterOrEqual(t, derp.ErrorCode(err), lowest, "got %v", err)
			require.LessOrEqual(t, derp.ErrorCode(err), highest, "got %v", err)
			require.Empty(t, session.collection.saved)
		})
	}

	// The service's own rules are bad requests
	refuses("not attached to a Stream", 400, 400, func(r *model.StreamSource) { r.StreamID = primitive.NilObjectID })
	refuses("credentials in the URL", 400, 400, func(r *model.StreamSource) { r.URL = "https://ben:secret@github.com/org/docs" })
	refuses("unparseable URL", 400, 400, func(r *model.StreamSource) { r.URL = "https://[::1" })

	// The schema's rules are client errors, with a code the schema library chooses rule by rule
	refuses("missing URL", 400, 499, func(r *model.StreamSource) { r.URL = "" })
	refuses("unsupported scheme", 400, 499, func(r *model.StreamSource) { r.URL = "file:///etc/emissary/docs" })
	refuses("unknown method", 400, 499, func(r *model.StreamSource) { r.Method = "FTP" })
}

// TestStreamSource_Save_NeverRepeatsPassword confirms that a refused address does not copy its
// password into the error, where it would reach the error log
func TestStreamSource_Save_NeverRepeatsPassword(t *testing.T) {

	for _, address := range []string{
		"https://ben:hunter2-SECRET@github.com/org/docs",
		"https://ben:hunter2-SECRET@[::1",
	} {
		service, session := newStreamSourceService()
		streamSource := validStreamSource()
		streamSource.URL = address

		err := service.Save(session, &streamSource, "Created")
		require.Error(t, err)

		encoded, marshalErr := json.Marshal(err)
		require.NoError(t, marshalErr)
		require.NotContains(t, string(encoded), "hunter2-SECRET")
	}
}

// TestStreamSource_LoadByStreamID finds the record attached to a Stream, and ignores deleted records
func TestStreamSource_LoadByStreamID(t *testing.T) {

	live := validStreamSource()

	deleted := validStreamSource()
	deleted.DeleteDate = 1

	service, session := newStreamSourceService(live, deleted)

	t.Run("attached", func(t *testing.T) {
		result := model.NewStreamSource()
		require.NoError(t, service.LoadByStreamID(session, live.StreamID, &result))
		require.Equal(t, live.StreamSourceID, result.StreamSourceID)
	})

	t.Run("deleted", func(t *testing.T) {
		result := model.NewStreamSource()
		require.True(t, derp.IsNotFound(service.LoadByStreamID(session, deleted.StreamID, &result)))
	})

	t.Run("no record", func(t *testing.T) {
		result := model.NewStreamSource()
		require.True(t, derp.IsNotFound(service.LoadByStreamID(session, primitive.NewObjectID(), &result)))
	})
}

// TestStreamSource_LoadByID finds a record by its identifier
func TestStreamSource_LoadByID(t *testing.T) {

	record := validStreamSource()
	service, session := newStreamSourceService(record)

	result := model.NewStreamSource()
	require.NoError(t, service.LoadByID(session, record.StreamSourceID, &result))
	require.Equal(t, record.URL, result.URL)

	require.True(t, derp.IsNotFound(service.LoadByID(session, primitive.NewObjectID(), &result)))
}

// TestStreamSource_Delete removes a record from every later lookup
func TestStreamSource_Delete(t *testing.T) {

	record := validStreamSource()
	service, session := newStreamSourceService(record)

	require.NoError(t, service.Delete(session, &record, "Removed"))

	result := model.NewStreamSource()
	require.True(t, derp.IsNotFound(service.LoadByID(session, record.StreamSourceID, &result)))

	count, err := service.Count(session, exp.All())
	require.NoError(t, err)
	require.Zero(t, count)
}

/******************************************
 * Status
 ******************************************/

// TestStreamSource_SetStatusLoading records that a sync has begun
func TestStreamSource_SetStatusLoading(t *testing.T) {

	record := validStreamSource()
	record.StatusMessage = "old message"
	service, session := newStreamSourceService(record)

	before := time.Now().Unix()
	require.NoError(t, service.SetStatusLoading(session, &record))
	after := time.Now().Unix()

	require.Equal(t, model.StreamSourceStatusLoading, record.Status)
	require.Empty(t, record.StatusMessage)
	require.GreaterOrEqual(t, record.LastSynced, before)
	require.LessOrEqual(t, record.LastSynced, after)
	require.Len(t, session.collection.saved, 1)
}

// TestStreamSource_SetStatusSuccess records the outcome and clears any previous message
func TestStreamSource_SetStatusSuccess(t *testing.T) {

	record := validStreamSource()
	record.Status = model.StreamSourceStatusLoading
	record.StatusMessage = "old message"
	record.LastSynced = 1_700_000_000

	service, session := newStreamSourceService(record)

	require.NoError(t, service.SetStatusSuccess(session, &record))

	require.Equal(t, model.StreamSourceStatusSuccess, record.Status)
	require.Empty(t, record.StatusMessage)
	require.Equal(t, int64(1_700_000_000), record.LastSynced, "success does not move LastSynced")
	require.Len(t, session.collection.saved, 1)
}

// TestStreamSource_SetStatusFailure records the reason the sync failed
func TestStreamSource_SetStatusFailure(t *testing.T) {

	record := validStreamSource()
	record.LastSynced = 1_700_000_000

	service, session := newStreamSourceService(record)

	require.NoError(t, service.SetStatusFailure(session, &record, "Repository not found"))

	require.Equal(t, model.StreamSourceStatusFailure, record.Status)
	require.Equal(t, "Repository not found", record.StatusMessage)
	require.Equal(t, int64(1_700_000_000), record.LastSynced, "a failure does not move LastSynced")
	require.Len(t, session.collection.saved, 1)
}

// TestStreamSource_SetStatusFailure_TruncatesMessage keeps a long message within its schema, as valid UTF-8
func TestStreamSource_SetStatusFailure_TruncatesMessage(t *testing.T) {

	record := validStreamSource()
	service, session := newStreamSourceService(record)

	// A three-byte character straddles the 1024-byte cut
	message := strings.Repeat("a", 1023) + "€" + strings.Repeat("b", 50)

	require.NoError(t, service.SetStatusFailure(session, &record, message))

	require.LessOrEqual(t, len(record.StatusMessage), 1024)
	require.True(t, utf8.ValidString(record.StatusMessage))
	require.Equal(t, strings.Repeat("a", 1023), record.StatusMessage)
}

// TestTruncateStatusMessage leaves a short message alone, and cuts a long one on a character boundary
func TestTruncateStatusMessage(t *testing.T) {

	require.Equal(t, "", truncateStatusMessage(""))
	require.Equal(t, "short", truncateStatusMessage("short"))

	exact := strings.Repeat("a", 1024)
	require.Equal(t, exact, truncateStatusMessage(exact))
	require.Equal(t, exact, truncateStatusMessage(exact+"b"))

	// Every cut through a four-byte character still leaves valid UTF-8
	for padding := 1020; padding <= 1024; padding++ {
		result := truncateStatusMessage(strings.Repeat("a", padding) + "🦖" + "tail")
		require.True(t, utf8.ValidString(result), "padding %d", padding)
		require.LessOrEqual(t, len(result), 1024)
	}
}

// TestValidateStreamSourceURL refuses an unreadable address or one that carries credentials
func TestValidateStreamSourceURL(t *testing.T) {

	require.NoError(t, validateStreamSourceURL("https://github.com/EmissarySocial/docs"))
	require.NoError(t, validateStreamSourceURL(""), "a missing address is the schema's to refuse")

	require.True(t, derp.IsBadRequest(validateStreamSourceURL("https://ben@github.com/org/docs")))
	require.True(t, derp.IsBadRequest(validateStreamSourceURL("https://ben:secret@github.com/org/docs")))
	require.True(t, derp.IsBadRequest(validateStreamSourceURL("https://[::1")))
}

// TestFactory_StreamSource registers the service, and its collection with the nightly recycler
func TestFactory_StreamSource(t *testing.T) {

	factory := Factory{}

	require.Same(t, &factory.streamSourceService, factory.StreamSource())
	require.True(t, slices.Contains(factory.Collections(), "StreamSource"),
		"a collection missing from Collections() keeps its deleted records forever")
}
