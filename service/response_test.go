package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests cover the two decisions that keep a User's reactions consistent: which existing
// Responses a new one displaces (conflictingResponses) and whether it displaces anything at all
// (responseIsUnchanged). They use a hand-built data.Collection fake for the same reason as
// collectionItem_test.go -- benpate/data-mock matches on the raw bson tag string and can't match
// the model's `,omitempty` fields.
//
// SetResponse itself is not exercised here: Save and Delete reach into the NewsFeed, Outbox,
// User, and ActivityStream services, which are concrete structs rather than interfaces.

/******************************************
 * responseStore -- an in-memory data.Collection that matches Responses on the fields
 * QueryByUserAndObject uses: userId, object, deleteDate.
 ******************************************/

// responseStore is an in-memory data.Collection of Responses, used by the tests in this file
type responseStore struct {
	records    []model.Response
	queryCount int // number of times Query has been called (proves the gate short-circuits before any read)
}

// Context implements the interface, returning a background context
func (c *responseStore) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *responseStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.NotFound("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *responseStore) Query(target any, criteria exp.Expression, _ ...option.Option) error {

	c.queryCount++

	result, ok := target.(*[]model.Response)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	for _, record := range c.records {
		if matchesResponse(criteria, record) {
			*result = append(*result, record)
		}
	}

	return nil
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *responseStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

// Load implements the data.Collection interface. Unused by these tests.
func (c *responseStore) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.NotFound("test", "unused")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *responseStore) Save(data.Object, string) error {
	return derp.NotFound("test", "unused")
}

// Delete implements the data.Collection interface. Unused by these tests.
func (c *responseStore) Delete(data.Object, string) error {
	return derp.NotFound("test", "unused")
}

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *responseStore) HardDelete(exp.Expression) error {
	return derp.NotFound("test", "unused")
}

// matchesResponse reports whether a record satisfies an equality criteria on
// userId / object / deleteDate.
func matchesResponse(criteria exp.Expression, record model.Response) bool {

	// Any unsupported field or operator conservatively counts as "no match".
	return criteria.Match(func(predicate exp.Predicate) bool {

		if predicate.Operator != exp.OperatorEqual {
			return false
		}

		switch predicate.Field {

		case "userId":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.UserID == value

		case "object":
			value, ok := predicate.Value.(string)
			return ok && record.Object == value

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && record.DeleteDate == int64(value)

		default:
			return false
		}
	})
}

// responseSession is a data.Session that hands out a single responseStore
type responseSession struct {
	store data.Collection
}

// Collection implements the data.Session interface, returning this stub's single collection
func (s responseSession) Collection(string) data.Collection { return s.store }

// Context implements the interface, returning a background context
func (s responseSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s responseSession) Close() {}

// newResponseService returns a Response service backed by the provided store
func newResponseService(store data.Collection) (*Response, responseSession) {
	service := NewResponse()
	return &service, responseSession{store: store}
}

// newTestResponse builds a persisted Response owned by the provided User.
func newTestResponse(userID primitive.ObjectID, object string, responseType string, content string) model.Response {

	response := model.NewResponse()
	response.UserID = userID
	response.Actor = "https://example.test/@user"
	response.Object = object
	response.Type = responseType
	response.Content = content

	return response
}

/******************************************
 * conflictingResponses
 ******************************************/

// A Like displaces an existing Dislike on the same object -- the mutual-exclusivity rule.
func TestResponse_ConflictingResponses_LikeDisplacesDislike(t *testing.T) {

	userID := primitive.NewObjectID()
	store := &responseStore{records: []model.Response{
		newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeDislike, ""),
	}}

	service, session := newResponseService(store)
	result, err := service.conflictingResponses(session, userID, "https://example.test/post", vocab.ActivityTypeLike)

	require.Nil(t, err)
	require.Len(t, result, 1)
	require.Equal(t, vocab.ActivityTypeDislike, result[0].Type)
}

// A Like leaves an existing Announce alone -- sharing and liking are not contradictory.
func TestResponse_ConflictingResponses_LikeIgnoresAnnounce(t *testing.T) {

	userID := primitive.NewObjectID()
	store := &responseStore{records: []model.Response{
		newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeAnnounce, ""),
	}}

	service, session := newResponseService(store)
	result, err := service.conflictingResponses(session, userID, "https://example.test/post", vocab.ActivityTypeLike)

	require.Nil(t, err)
	require.Empty(t, result)
}

// A Like displaces a previous Like, so a repeat replaces rather than duplicates.
func TestResponse_ConflictingResponses_LikeDisplacesLike(t *testing.T) {

	userID := primitive.NewObjectID()
	store := &responseStore{records: []model.Response{
		newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeLike, ""),
	}}

	service, session := newResponseService(store)
	result, err := service.conflictingResponses(session, userID, "https://example.test/post", vocab.ActivityTypeLike)

	require.Nil(t, err)
	require.Len(t, result, 1)
	require.Equal(t, vocab.ActivityTypeLike, result[0].Type)
}

// The pre-fix corruption (a Like AND a Dislike coexisting) is reported in full, so that
// SetResponse clears both and the contradiction is repaired on the next reaction.
func TestResponse_ConflictingResponses_ReportsExistingContradiction(t *testing.T) {

	userID := primitive.NewObjectID()
	store := &responseStore{records: []model.Response{
		newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeLike, ""),
		newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeDislike, ""),
		newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeAnnounce, ""),
	}}

	service, session := newResponseService(store)
	result, err := service.conflictingResponses(session, userID, "https://example.test/post", vocab.ActivityTypeLike)

	require.Nil(t, err)
	require.Len(t, result, 2)
}

// Reactions belonging to other Users, or to other Objects, are never displaced.
func TestResponse_ConflictingResponses_ScopedToUserAndObject(t *testing.T) {

	userID := primitive.NewObjectID()
	store := &responseStore{records: []model.Response{
		newTestResponse(primitive.NewObjectID(), "https://example.test/post", vocab.ActivityTypeLike, ""),
		newTestResponse(userID, "https://example.test/other", vocab.ActivityTypeLike, ""),
	}}

	service, session := newResponseService(store)
	result, err := service.conflictingResponses(session, userID, "https://example.test/post", vocab.ActivityTypeLike)

	require.Nil(t, err)
	require.Empty(t, result)
}

// A soft-deleted Response is invisible, matching the notDeleted() read semantics.
func TestResponse_ConflictingResponses_IgnoresDeleted(t *testing.T) {

	userID := primitive.NewObjectID()
	deleted := newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeLike, "")
	deleted.DeleteDate = 1

	store := &responseStore{records: []model.Response{deleted}}

	service, session := newResponseService(store)
	result, err := service.conflictingResponses(session, userID, "https://example.test/post", vocab.ActivityTypeLike)

	require.Nil(t, err)
	require.Empty(t, result)
}

// A database failure is reported, never silently read as "nothing conflicts" -- which would
// let SetResponse write a duplicate on top of a reaction it simply failed to see.
func TestResponse_ConflictingResponses_ReportsQueryError(t *testing.T) {

	service, session := newResponseService(&brokenResponseStore{})
	result, err := service.conflictingResponses(session, primitive.NewObjectID(), "https://example.test/post", vocab.ActivityTypeLike)

	require.NotNil(t, err)
	require.Nil(t, result)
}

// brokenResponseStore is a responseStore whose Query always fails.
type brokenResponseStore struct {
	responseStore
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *brokenResponseStore) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "database is on fire")
}

/******************************************
 * responseIsUnchanged
 ******************************************/

// The idempotency guard: repeating an identical Like changes nothing.
func TestResponse_IsUnchanged_RepeatedLike(t *testing.T) {

	existing := []model.Response{
		newTestResponse(primitive.NewObjectID(), "https://example.test/post", vocab.ActivityTypeLike, ""),
	}

	require.True(t, responseIsUnchanged(existing, vocab.ActivityTypeLike, ""))
}

// A first-time reaction has work to do.
func TestResponse_IsUnchanged_NoExistingResponse(t *testing.T) {
	require.False(t, responseIsUnchanged([]model.Response{}, vocab.ActivityTypeLike, ""))
	require.False(t, responseIsUnchanged(nil, vocab.ActivityTypeLike, ""))
}

// Switching from Dislike to Like is a real change.
func TestResponse_IsUnchanged_SwitchingType(t *testing.T) {

	existing := []model.Response{
		newTestResponse(primitive.NewObjectID(), "https://example.test/post", vocab.ActivityTypeDislike, ""),
	}

	require.False(t, responseIsUnchanged(existing, vocab.ActivityTypeLike, ""))
}

// Same type, different content (e.g. changing an emoji) is a real change.
func TestResponse_IsUnchanged_DifferentContent(t *testing.T) {

	existing := []model.Response{
		newTestResponse(primitive.NewObjectID(), "https://example.test/post", vocab.ActivityTypeLike, "👍"),
	}

	require.False(t, responseIsUnchanged(existing, vocab.ActivityTypeLike, "❤️"))
	require.True(t, responseIsUnchanged(existing, vocab.ActivityTypeLike, "👍"))
}

// Two coexisting reactions are always a contradiction to resolve, never a no-op -- even when one
// of them is the exact reaction being set. This is what makes SetResponse repair pre-fix data.
func TestResponse_IsUnchanged_ContradictionIsNeverUnchanged(t *testing.T) {

	userID := primitive.NewObjectID()
	existing := []model.Response{
		newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeLike, ""),
		newTestResponse(userID, "https://example.test/post", vocab.ActivityTypeDislike, ""),
	}

	require.False(t, responseIsUnchanged(existing, vocab.ActivityTypeLike, ""))
}

/******************************************
 * validateReactionTarget
 ******************************************/

// loaderThatFails returns a loadDocument fake that always fails, and records whether it was called.
func loaderThatFails(called *bool) func(string) (streams.Document, error) {
	return func(string) (streams.Document, error) {
		*called = true
		return streams.NilDocument(), derp.NotFound("test", "object does not exist")
	}
}

// A malformed / non-http(s) URL is rejected WITHOUT ever hitting the network -- the cheap syntactic
// gate must short-circuit before the loader runs.
func TestResponse_ValidateReactionTarget_InvalidURL(t *testing.T) {

	loaderCalled := false
	service, _ := newResponseService(&responseStore{})
	service.loadDocument = loaderThatFails(&loaderCalled)

	for _, url := range []string{"", "not-a-url", "javascript:alert(1)", "/relative/path", "ftp://example.test/x"} {
		object, err := service.validateReactionTarget(url)

		require.NotNil(t, err, "url %q should be rejected", url)
		require.True(t, object.IsNil())
		require.False(t, loaderCalled, "url %q must be rejected before the loader is called", url)
	}
}

// A well-formed URL that does not resolve to a real object is rejected -- this is the nonexistent
// local object / arbitrary external URL case from the bug report.
func TestResponse_ValidateReactionTarget_Unresolvable(t *testing.T) {

	loaderCalled := false
	service, _ := newResponseService(&responseStore{})
	service.loadDocument = loaderThatFails(&loaderCalled)

	object, err := service.validateReactionTarget("https://example.com/000000000000000000000000-does-not-exist")

	require.NotNil(t, err)
	require.True(t, object.IsNil())
	require.True(t, loaderCalled, "a valid URL must be resolved through the loader")
}

// A well-formed URL that resolves is accepted, and the loaded document is returned for reuse.
func TestResponse_ValidateReactionTarget_Valid(t *testing.T) {

	service, _ := newResponseService(&responseStore{})
	service.loadDocument = func(url string) (streams.Document, error) {
		return streams.NewDocument(mapof.Any{"id": url, "attributedTo": "https://example.com/@author"}), nil
	}

	object, err := service.validateReactionTarget("https://example.com/post")

	require.Nil(t, err)
	require.Equal(t, "https://example.com/post", object.ID())
	require.Equal(t, "https://example.com/@author", object.AttributedTo().ID())
}

/******************************************
 * SetResponse -- validation gate
 ******************************************/

// SetResponse rejects a malformed target and, crucially, writes NOTHING: the gate runs before the
// conflict query, so the store is never even read.
func TestResponse_SetResponse_RejectsInvalidURL(t *testing.T) {

	loaderCalled := false
	store := &responseStore{}
	service, session := newResponseService(store)
	service.loadDocument = loaderThatFails(&loaderCalled)

	user := model.NewUser()

	err := service.SetResponse(session, &user, "not-a-url", vocab.ActivityTypeLike, "")

	require.NotNil(t, err)
	require.False(t, loaderCalled)
	require.Zero(t, store.queryCount, "a rejected target must not reach the conflict query")
}

// SetResponse rejects a well-formed but unresolvable target (nonexistent local object or arbitrary
// external URL) and, again, writes nothing.
func TestResponse_SetResponse_RejectsUnresolvableTarget(t *testing.T) {

	loaderCalled := false
	store := &responseStore{}
	service, session := newResponseService(store)
	service.loadDocument = loaderThatFails(&loaderCalled)

	user := model.NewUser()

	err := service.SetResponse(session, &user, "https://evil.example.com/anything", vocab.ActivityTypeLike, "")

	require.NotNil(t, err)
	require.True(t, loaderCalled)
	require.Zero(t, store.queryCount, "a rejected target must not reach the conflict query")
}
