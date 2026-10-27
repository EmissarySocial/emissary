package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newTestObjectServices returns an Object service wired to an Outbox2, both on example.com
func newTestObjectServices() (*Object, *Outbox2) {
	locator := Locator{host: "https://example.com"}
	outbox2 := Outbox2{locator: &locator, host: "https://example.com"}
	object := Object{host: "https://example.com", locatorService: &locator, outbox2Service: &outbox2}
	return &object, &outbox2
}

// newTestNoteActivity returns a Create activity for a note from userID, addressed to "to"
func newTestNoteActivity(userID primitive.ObjectID, to ...any) streams.Document {
	return streams.NewDocument(mapof.Any{
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyID:    "urn:uuid:client-chosen-activity-id",
		vocab.PropertyActor: "https://example.com/@" + userID.Hex(),
		vocab.PropertyTo:    to,
		vocab.PropertyObject: map[string]any{
			vocab.PropertyType:    vocab.ObjectTypeNote,
			vocab.PropertyID:      "urn:uuid:client-chosen-object-id",
			vocab.PropertyContent: "@bob hello",
			vocab.PropertyTo:      to,
		},
	})
}

// TestObject_CreateAddressed confirms an addressed note is saved for its recipients only, with
// server-assigned ids, and put into the sender's outbox for delivery
func TestObject_CreateAddressed(t *testing.T) {

	objectService, _ := newTestObjectServices()
	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	userID := primitive.NewObjectID()
	activity := newTestNoteActivity(userID, "https://remote.example/@bob")

	objectURL, activityURL, err := objectService.CreateAddressed(session, userID, activity)
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(objectURL, "https://example.com/@"+userID.Hex()+"/pub/objects/"), objectURL)
	require.Equal(t, objectURL, activity.Object().ID(), "the delivered activity carries the server's object id")
	require.Equal(t, activityURL, activity.ID(), "the delivered activity carries the server's activity id")

	// The Object is readable only by its recipients
	object := model.NewObject()
	require.NoError(t, session.Collection("Object").Load(exp.Equal("userId", userID), &object))
	require.Equal(t, objectURL, object.Value[vocab.PropertyID])
	require.Equal(t, []string{"https://remote.example/@bob"}, []string(object.Permissions))

	// The activity is in the sender's outbox, addressed to the same recipient
	item := model.NewOutboxItem()
	require.NoError(t, session.Collection("Outbox2").Load(exp.Equal("actorId", userID), &item))
	require.Equal(t, activityURL, item.URL)
	require.True(t, item.Recipients.Contains("https://remote.example/@bob"))
}

// TestObject_CreateAddressed_RefusesPublic confirms a public note is refused before anything is saved
func TestObject_CreateAddressed_RefusesPublic(t *testing.T) {

	objectService, _ := newTestObjectServices()
	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	userID := primitive.NewObjectID()
	activity := newTestNoteActivity(userID, vocab.NamespaceActivityStreamsPublic)

	_, _, err = objectService.CreateAddressed(session, userID, activity)
	require.Error(t, err)

	count, err := session.Collection("Object").Count(exp.All())
	require.NoError(t, err)
	require.Zero(t, count, "a refused public note must not leave an Object behind")
}

// TestOutbox2_PutUserActivity_RefusesOtherActor confirms a User cannot send as someone else
func TestOutbox2_PutUserActivity_RefusesOtherActor(t *testing.T) {

	_, outbox2 := newTestObjectServices()
	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	activity := newTestNoteActivity(primitive.NewObjectID(), "https://remote.example/@bob")

	_, err = outbox2.PutUserActivity(session, primitive.NewObjectID(), activity)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))

	count, err := session.Collection("Outbox2").Count(exp.All())
	require.NoError(t, err)
	require.Zero(t, count)
}
