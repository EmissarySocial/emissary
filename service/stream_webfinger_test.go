package service

import (
	"context"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/set"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/digit"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests cover the handles a Stream actor publishes through WebFinger (BUG-98): the subject
// must be the value the actor document carries as preferredUsername, every acct: form the record
// advertises must load the same Stream, and a Stream that is not an actor is a 404. The fakes
// mirror user_webfinger_test.go. The Template service is built directly, because Load reads
// nothing but its in-memory map.

/******************************************
 * In-Memory Fakes
 ******************************************/

// streamCollection is an in-memory data.Collection holding model.Stream records, matched on the
// fields that LoadByID, LoadByToken, and notDeleted build: _id, token, and deleteDate.
type streamCollection struct {
	records   []model.Stream
	loads     int   // Number of Load calls, so a test can assert the collection was never consulted
	loadError error // When set, every Load fails with this error
}

// Context implements the interface, returning a background context
func (c *streamCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (c *streamCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query implements the data.Collection interface. Unused by these tests.
func (c *streamCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (c *streamCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load copies the first stored Stream that satisfies the criteria into the target.
func (c *streamCollection) Load(criteria exp.Expression, target data.Object, options ...option.Option) error {

	c.loads++

	if c.loadError != nil {
		return c.loadError
	}

	stream, ok := target.(*model.Stream)

	if !ok {
		return derp.Internal("test", "unexpected target type")
	}

	for _, record := range c.records {
		if matchesStream(criteria, record, isCaseSensitive(options)) {
			*stream = record
			return nil
		}
	}

	return derp.NotFound("test", "not found")
}

// Save implements the data.Collection interface. Unused by these tests.
func (c *streamCollection) Save(data.Object, string) error { return derp.Internal("test", "unused") }

// Delete implements the data.Collection interface. Unused by these tests.
func (c *streamCollection) Delete(data.Object, string) error { return derp.Internal("test", "unused") }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (c *streamCollection) HardDelete(exp.Expression) error { return derp.Internal("test", "unused") }

// isCaseSensitive reports whether the query options ask for a case-sensitive match, which is the default
func isCaseSensitive(options []option.Option) bool {

	for _, item := range options {
		if typed, ok := item.(option.CaseSensitiveOption); ok {
			return typed.CaseSensitive()
		}
	}

	return true
}

// matchesStream reports whether the stored Stream satisfies a criteria on _id, token, and deleteDate.
func matchesStream(criteria exp.Expression, record model.Stream, caseSensitive bool) bool {

	return criteria.Match(func(predicate exp.Predicate) bool {

		if predicate.Operator != exp.OperatorEqual {
			return false
		}

		switch predicate.Field {

		case "_id":
			value, ok := predicate.Value.(primitive.ObjectID)
			return ok && record.StreamID == value

		case "token":
			value, ok := predicate.Value.(string)

			if !ok {
				return false
			}

			if caseSensitive {
				return record.Token == value
			}

			return strings.EqualFold(record.Token, value)

		case "deleteDate":
			value, ok := predicate.Value.(int)
			return ok && record.DeleteDate == int64(value)

		default:
			return false
		}
	})
}

// webfingerSession routes the User and Stream collections to their in-memory fakes. A test may
// also attach a SearchQuery collection; every other name answers with an empty Stream fake.
type webfingerSession struct {
	users         *userCollection
	streams       *streamCollection
	searchQueries data.Collection
}

// Collection implements the data.Session interface
func (s webfingerSession) Collection(name string) data.Collection {

	switch name {

	case "User":
		return s.users

	case "Stream":
		return s.streams

	case "SearchQuery":
		if s.searchQueries != nil {
			return s.searchQueries
		}
	}

	return &streamCollection{}
}

// Context implements the interface, returning a background context
func (s webfingerSession) Context() context.Context { return context.Background() }

// Close implements the interface. The stub holds no resources to release.
func (s webfingerSession) Close() {}

// actorTemplate returns a Template whose Streams federate as actors
func actorTemplate(templateID string) model.Template {
	template := model.NewTemplate(templateID, nil)
	template.Actor = model.StreamActor{SocialRole: vocab.ActorTypeApplication}
	return template
}

// newActorStream returns a Stream that uses the named Template, with the actor id its Save would compute
func newActorStream(templateID string, token string) model.Stream {

	stream := model.NewStream()
	stream.TemplateID = templateID
	stream.URL = "https://example.com/" + stream.StreamID.Hex()

	if token != "" {
		stream.Token = token
	}

	return stream
}

// newStreamWebFingerService returns a Stream service backed by the provided Templates and Streams
func newStreamWebFingerService(templates []model.Template, streams ...model.Stream) (*Stream, webfingerSession) {

	templateService := &Template{templates: set.Map[model.Template]{}}

	for _, template := range templates {
		templateService.templates[template.TemplateID] = template
	}

	service := &Stream{host: "https://example.com", templateService: templateService}
	session := webfingerSession{users: &userCollection{}, streams: &streamCollection{records: streams}}

	return service, session
}

// selfLink returns the href of the resource's rel=self link, or "" when there is none
func selfLink(resource digit.Resource) string {

	for _, link := range resource.Links {
		if link.RelationType == digit.RelationTypeSelf {
			return link.Href
		}
	}

	return ""
}

/******************************************
 * Tests
 ******************************************/

// TestStreamWebFinger_SubjectIsTheHandle confirms that a token which qualifies as a handle is the
// subject, with the StreamID forms and the token URL as aliases, and the actor id as the self link.
func TestStreamWebFinger_SubjectIsTheHandle(t *testing.T) {

	stream := newActorStream("group", "my-article")
	streamID := stream.StreamID.Hex()
	service, session := newStreamWebFingerService([]model.Template{actorTemplate("group")}, stream)

	resource, err := service.WebFinger(session, "my-article")

	require.NoError(t, err)
	require.Equal(t, "acct:my-article@example.com", resource.Subject)
	require.Equal(t, []string{
		"acct:" + streamID + "@example.com",
		"https://example.com/my-article",
		"https://example.com/" + streamID,
	}, resource.Aliases)
	require.Equal(t, stream.ActivityPubURL(), selfLink(resource))
}

// TestStreamWebFinger_DefaultTokenEmitsNoDuplicates confirms that a Stream still carrying its
// default token (the StreamID) advertises each form once.
func TestStreamWebFinger_DefaultTokenEmitsNoDuplicates(t *testing.T) {

	stream := newActorStream("group", "")
	streamID := stream.StreamID.Hex()
	service, session := newStreamWebFingerService([]model.Template{actorTemplate("group")}, stream)

	resource, err := service.WebFinger(session, streamID)

	require.NoError(t, err)
	require.Equal(t, "acct:"+streamID+"@example.com", resource.Subject)
	require.Equal(t, []string{"https://example.com/" + streamID}, resource.Aliases)
}

// TestStreamWebFinger_UnsafeTokenFallsBackToID confirms that a token outside Mastodon's username
// grammar is not published as a handle, while its page URL still resolves.
func TestStreamWebFinger_UnsafeTokenFallsBackToID(t *testing.T) {

	stream := newActorStream("group", "café")
	streamID := stream.StreamID.Hex()
	service, session := newStreamWebFingerService([]model.Template{actorTemplate("group")}, stream)

	resource, err := service.WebFinger(session, "café")

	require.NoError(t, err)
	require.Equal(t, "acct:"+streamID+"@example.com", resource.Subject)
	require.Equal(t, []string{
		"https://example.com/café",
		"https://example.com/" + streamID,
	}, resource.Aliases)
}

// TestStreamWebFinger_LoadsByEitherForm confirms that the token and the StreamID name the same resource
func TestStreamWebFinger_LoadsByEitherForm(t *testing.T) {

	stream := newActorStream("group", "my-article")
	service, session := newStreamWebFingerService([]model.Template{actorTemplate("group")}, stream)

	byToken, err := service.WebFinger(session, "my-article")
	require.NoError(t, err)

	byID, err := service.WebFinger(session, stream.StreamID.Hex())
	require.NoError(t, err)

	require.Equal(t, byToken, byID)
}

// TestStreamWebFinger_NotAnActorIs404 confirms that a Stream whose Template defines no actor is
// reported as not found, not as a bad request (RFC 7033 §4.5).
func TestStreamWebFinger_NotAnActorIs404(t *testing.T) {

	stream := newActorStream("page", "about")
	service, session := newStreamWebFingerService([]model.Template{model.NewTemplate("page", nil)}, stream)

	_, err := service.WebFinger(session, "about")

	require.Error(t, err)
	require.Equal(t, 404, derp.ErrorCode(err))
}

// TestStreamWebFinger_MissingStreamIs404 confirms that an unknown token is a 404 by either form
func TestStreamWebFinger_MissingStreamIs404(t *testing.T) {

	service, session := newStreamWebFingerService([]model.Template{actorTemplate("group")})

	for _, token := range []string{"nothing", primitive.NewObjectID().Hex()} {
		_, err := service.WebFinger(session, token)
		require.Error(t, err, "loading %q", token)
		require.Equal(t, 404, derp.ErrorCode(err), "loading %q", token)
	}
}

// TestStreamWebFinger_LoadErrorIsNotA404 confirms that a database failure is not reported as "not found"
func TestStreamWebFinger_LoadErrorIsNotA404(t *testing.T) {

	service, session := newStreamWebFingerService([]model.Template{actorTemplate("group")})
	session.streams.loadError = derp.Internal("test", "database unavailable")

	_, err := service.WebFinger(session, "my-article")

	require.Error(t, err)
	require.Equal(t, 500, derp.ErrorCode(err))
}

// TestStreamWebFinger_MatchesActorDocument is the invariant behind BUG-98: the WebFinger subject is
// exactly acct:<preferredUsername>@<host>, for a token that qualifies as a handle and for one that
// does not, because both values come from Stream.ActivityPubUsername.
func TestStreamWebFinger_MatchesActorDocument(t *testing.T) {

	template := actorTemplate("group")

	for _, token := range []string{"my-article", "café", ""} {

		stream := newActorStream("group", token)
		service, session := newStreamWebFingerService([]model.Template{template}, stream)

		resource, err := service.WebFinger(session, stream.Token)
		require.NoError(t, err, "token %q", token)

		preferredUsername := template.Actor.JSONLD(&stream)[vocab.PropertyPreferredUsername]
		require.Equal(t, "acct:"+preferredUsername.(string)+"@example.com", resource.Subject, "token %q", token)
	}
}
