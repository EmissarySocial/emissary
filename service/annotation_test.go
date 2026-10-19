package service

import (
	"context"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Fakes
 ******************************************/

// annotationClient is a streams.Client that returns one scripted document, or one scripted error.
type annotationClient struct {
	document map[string]any
	err      error
}

// SetRootClient implements the streams.Client interface
func (client annotationClient) SetRootClient(streams.Client) {}

// Load returns the scripted document or error
func (client annotationClient) Load(string, ...any) (streams.Document, error) {

	if client.err != nil {
		return streams.NilDocument(), client.err
	}

	return streams.NewDocument(client.document), nil
}

// Save implements the streams.Client interface
func (client annotationClient) Save(streams.Document) error { return nil }

// Delete implements the streams.Client interface
func (client annotationClient) Delete(string) error { return nil }

// annotationStore is a data.Collection fake whose first Save can lose a race to a stored winner.
type annotationStore struct {
	winner    *model.Annotation // when set, the first Save fails with a Conflict and Load returns this
	late      bool              // when set, Load misses the winner, as if it arrived after the check
	saveErr   error             // when set, every Save fails with this error
	saveCalls int
	saved     []model.Annotation
	criteria  exp.Expression
}

// Context implements the data.Collection interface
func (store *annotationStore) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface. Unused by these tests.
func (store *annotationStore) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, nil
}

// Query records the criteria it was asked for
func (store *annotationStore) Query(_ any, criteria exp.Expression, _ ...option.Option) error {
	store.criteria = criteria
	return nil
}

// Iterator implements the data.Collection interface. Unused by these tests.
func (store *annotationStore) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.NotFound("test", "unused")
}

// Load returns the winner of the race, if there is one
func (store *annotationStore) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {

	if store.winner == nil || store.late {
		return derp.NotFound("test", "not found")
	}

	if annotation, ok := target.(*model.Annotation); ok {
		*annotation = *store.winner
		return nil
	}

	return derp.NotFound("test", "not found")
}

// Save records the object, failing the first call with a Conflict when a winner is scripted
func (store *annotationStore) Save(object data.Object, _ string) error {

	store.saveCalls++

	if store.saveErr != nil {
		return store.saveErr
	}

	if store.winner != nil && store.saveCalls == 1 {
		return derp.Wrap(derp.Internal("test", "E11000"), "test", "duplicate", derp.WithConflict())
	}

	if annotation, ok := object.(*model.Annotation); ok {
		store.saved = append(store.saved, *annotation)
	}

	return nil
}

// Delete implements the data.Collection interface. Unused by these tests.
func (store *annotationStore) Delete(data.Object, string) error { return nil }

// HardDelete implements the data.Collection interface. Unused by these tests.
func (store *annotationStore) HardDelete(exp.Expression) error { return nil }

// annotationSession is a data.Session whose Collection always returns the fake store.
type annotationSession struct {
	data.Session
	store data.Collection
}

// Collection implements the data.Session interface
func (session annotationSession) Collection(string) data.Collection { return session.store }

// newTestAnnotation returns an Annotation of one URL, owned by a new User
func newTestAnnotation() model.Annotation {
	annotation := model.NewAnnotation()
	annotation.UserID = primitive.NewObjectID()
	annotation.URL = "https://example.test/notes/1"
	return annotation
}

/******************************************
 * save
 ******************************************/

func TestAnnotation_Save_DescribesTheDocument(t *testing.T) {

	store := &annotationStore{}
	service := NewAnnotation()
	annotation := newTestAnnotation()

	client := annotationClient{document: map[string]any{
		"id":      annotation.URL,
		"type":    "Note",
		"content": "<p>Hello <b>world</b></p>",
		"attributedTo": map[string]any{
			"id":   "https://example.test/@alice",
			"type": "Person",
			"icon": "https://example.test/alice.png",
		},
	}}

	require.NoError(t, service.save(annotationSession{store: store}, client, &annotation, "test"))
	require.Len(t, store.saved, 1)
	require.Equal(t, model.AnnotationTypeDocument, store.saved[0].Type)
	require.Equal(t, "Hello world", store.saved[0].Name)
	require.Equal(t, "https://example.test/alice.png", store.saved[0].Icon)
	require.False(t, store.saved[0].IsActor())
}

func TestAnnotation_Save_NewAnnotationNeedsTheDocument(t *testing.T) {

	store := &annotationStore{}
	service := NewAnnotation()
	annotation := newTestAnnotation()

	client := annotationClient{err: derp.NotFound("test", "gone")}

	require.Error(t, service.save(annotationSession{store: store}, client, &annotation, "test"))
	require.Empty(t, store.saved)
}

func TestAnnotation_Save_ExistingAnnotationKeepsItsDescription(t *testing.T) {

	store := &annotationStore{}
	service := NewAnnotation()

	annotation := newTestAnnotation()
	annotation.CreateDate = 1
	annotation.Type = model.AnnotationTypeDocument
	annotation.Name = "Stored Name"
	annotation.Icon = "https://example.test/stored.png"
	annotation.Content = "an edited note"

	client := annotationClient{err: derp.NotFound("test", "gone")}

	require.NoError(t, service.save(annotationSession{store: store}, client, &annotation, "test"))
	require.Len(t, store.saved, 1)
	require.Equal(t, model.AnnotationTypeDocument, store.saved[0].Type)
	require.Equal(t, "Stored Name", store.saved[0].Name)
	require.Equal(t, "https://example.test/stored.png", store.saved[0].Icon)
	require.Equal(t, "an edited note", store.saved[0].Content)
}

func TestAnnotation_Save_TypeIsStampedOnlyOnCreate(t *testing.T) {

	service := NewAnnotation()
	actor := annotationClient{document: map[string]any{"id": "https://example.test/@alice", "type": "Person", "name": "Alice"}}

	// A new Annotation of an actor is stamped ACTOR
	created := newTestAnnotation()
	store := &annotationStore{}
	require.NoError(t, service.save(annotationSession{store: store}, actor, &created, "test"))
	require.Equal(t, model.AnnotationTypeActor, store.saved[0].Type)

	// A new Annotation keeps the type its linking page already set
	linked := newTestAnnotation()
	linked.Type = model.AnnotationTypeDocument
	store = &annotationStore{}
	require.NoError(t, service.save(annotationSession{store: store}, actor, &linked, "test"))
	require.Equal(t, model.AnnotationTypeDocument, store.saved[0].Type)

	// An existing Annotation keeps its type, whatever the live document now says
	existing := newTestAnnotation()
	existing.CreateDate = 1
	existing.Type = model.AnnotationTypeDocument
	store = &annotationStore{}
	require.NoError(t, service.save(annotationSession{store: store}, actor, &existing, "test"))
	require.Equal(t, model.AnnotationTypeDocument, store.saved[0].Type)
}

func TestAnnotation_Save_RejectsAnUnknownType(t *testing.T) {

	store := &annotationStore{}
	service := NewAnnotation()

	annotation := newTestAnnotation()
	annotation.CreateDate = 1
	annotation.Type = "Note"

	client := annotationClient{err: derp.NotFound("test", "gone")}

	require.Error(t, service.save(annotationSession{store: store}, client, &annotation, "test"))
	require.Empty(t, store.saved)
}

func TestAnnotation_Save_ConflictFoldsOntoTheWinner(t *testing.T) {

	annotation := newTestAnnotation()
	annotation.Content = "my note"

	winner := model.NewAnnotation()
	winner.UserID = annotation.UserID
	winner.URL = annotation.URL
	winner.CreateDate = 42

	store := &annotationStore{winner: &winner}
	service := NewAnnotation()
	client := annotationClient{document: map[string]any{"id": annotation.URL, "type": "Person", "name": "Alice"}}

	require.NoError(t, service.save(annotationSession{store: store}, client, &annotation, "test"))
	require.Equal(t, 2, store.saveCalls, "one lost insert, then one update")
	require.Len(t, store.saved, 1)
	require.Equal(t, winner.AnnotationID, store.saved[0].AnnotationID)
	require.Equal(t, int64(42), store.saved[0].CreateDate)
	require.Equal(t, "my note", store.saved[0].Content, "the incoming note wins")
}

func TestAnnotation_Save_ErrorsNameTheRecordNotItsNote(t *testing.T) {

	const note = "my private note about this account"
	service := NewAnnotation()
	client := annotationClient{document: map[string]any{"id": "https://example.test/notes/1", "type": "Person", "name": "Alice"}}

	t.Run("validation", func(t *testing.T) {
		annotation := newTestAnnotation()
		annotation.Content = note + strings.Repeat("x", 10000)
		err := service.save(annotationSession{store: &annotationStore{}}, client, &annotation, "test")
		require.Error(t, err)
		secretcheck.RequireAbsent(t, err, note)
	})

	t.Run("database", func(t *testing.T) {
		annotation := newTestAnnotation()
		annotation.Content = note
		store := &annotationStore{saveErr: derp.Internal("test", "database is down")}
		err := service.save(annotationSession{store: store}, client, &annotation, "test")
		require.Error(t, err)
		secretcheck.RequireAbsent(t, err, note)
	})
}

/******************************************
 * saveImported
 ******************************************/

func TestAnnotation_SaveImported_FailsWhenTheURLIsAnnotated(t *testing.T) {

	annotation := newTestAnnotation()
	annotation.Content = "imported note"

	existing := model.NewAnnotation()
	existing.UserID = annotation.UserID
	existing.URL = annotation.URL
	existing.Content = "my own note"

	store := &annotationStore{winner: &existing}
	service := NewAnnotation()
	client := annotationClient{document: map[string]any{"id": annotation.URL, "type": "Person", "name": "Alice"}}

	err := service.saveImported(annotationSession{store: store}, client, &annotation, "test")
	require.True(t, derp.IsConflict(err), "a duplicate import fails as a conflict")
	require.Zero(t, store.saveCalls, "nothing is written, so the import's transaction survives")
}

func TestAnnotation_SaveImported_LostRaceDoesNotFold(t *testing.T) {

	annotation := newTestAnnotation()
	annotation.Content = "imported note"

	winner := model.NewAnnotation()
	winner.UserID = annotation.UserID
	winner.URL = annotation.URL

	store := &annotationStore{winner: &winner, late: true}
	service := NewAnnotation()
	client := annotationClient{document: map[string]any{"id": annotation.URL, "type": "Person", "name": "Alice"}}

	err := service.saveImported(annotationSession{store: store}, client, &annotation, "test")
	require.True(t, derp.IsConflict(err), "the lost insert is returned, not folded")
	require.Equal(t, 1, store.saveCalls)
	require.Empty(t, store.saved, "the winner's note is never overwritten")
}

func TestAnnotation_SaveImported_SavesANewURL(t *testing.T) {

	annotation := newTestAnnotation()
	store := &annotationStore{}
	service := NewAnnotation()
	client := annotationClient{document: map[string]any{"id": annotation.URL, "type": "Person", "name": "Alice"}}

	require.NoError(t, service.saveImported(annotationSession{store: store}, client, &annotation, "test"))
	require.Len(t, store.saved, 1)
}

/******************************************
 * Queries
 ******************************************/

func TestAnnotation_QueryByUser_FiltersOnUserID(t *testing.T) {

	store := &annotationStore{}
	service := NewAnnotation()
	userID := primitive.NewObjectID()

	_, err := service.QueryByUser(annotationSession{store: store}, userID)
	require.NoError(t, err)

	require.Contains(t, store.criteria.Fields(), "userId")
	require.NotContains(t, store.criteria.Fields(), "_id")

	// The userId predicate holds the User's own ID
	require.True(t, store.criteria.Match(func(predicate exp.Predicate) bool {
		return predicate.Field != "userId" || predicate.Value == userID
	}))
}

/******************************************
 * Descriptions
 ******************************************/

func TestAnnotationName(t *testing.T) {

	// A named document keeps its name
	require.Equal(t, "Title", annotationName(streams.NewDocument(map[string]any{"type": "Article", "name": "Title", "content": "body"})))

	// A Note falls back to its summary, then its content
	require.Equal(t, "CW", annotationName(streams.NewDocument(map[string]any{"type": "Note", "summary": "CW", "content": "body"})))
	require.Equal(t, "body", annotationName(streams.NewDocument(map[string]any{"type": "Note", "content": "<p>body</p>"})))

	// HTML entities are decoded, so the template does not escape them a second time
	require.Equal(t, "Here's another reply", annotationName(streams.NewDocument(map[string]any{"type": "Note", "content": "<p>Here&#39;s another reply</p>"})))
	require.Equal(t, "Tom & Jerry", annotationName(streams.NewDocument(map[string]any{"type": "Note", "summary": "Tom &amp; Jerry"})))

	// An actor with no name falls back to its username
	require.Equal(t, "@alice@example.test", annotationName(streams.NewDocument(map[string]any{
		"id":                "https://example.test/@alice",
		"type":              "Person",
		"preferredUsername": "alice",
	})))

	// A name longer than the schema allows is shortened
	long := strings.Repeat("é", annotationMaxLength+10)
	require.Len(t, []rune(annotationName(streams.NewDocument(map[string]any{"type": "Article", "name": long}))), annotationMaxLength)
}

func TestAnnotationIcon(t *testing.T) {

	// A document's own icon wins
	require.Equal(t, "https://example.test/own.png", annotationIcon(streams.NewDocument(map[string]any{
		"type":         "Article",
		"icon":         "https://example.test/own.png",
		"attributedTo": map[string]any{"type": "Person", "icon": "https://example.test/author.png"},
	})))

	// An actor never borrows an icon
	require.Equal(t, "", annotationIcon(streams.NewDocument(map[string]any{
		"type":         "Person",
		"attributedTo": map[string]any{"type": "Person", "icon": "https://example.test/author.png"},
	})))

	// A URL too long to store is left out
	require.Equal(t, "", annotationIcon(streams.NewDocument(map[string]any{
		"type": "Note",
		"icon": "https://example.test/" + strings.Repeat("a", annotationMaxLength),
	})))
}
