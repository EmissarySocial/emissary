package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestObject_ErrorsCarryNoValue verifies that every error Object writes names the Object by its
// id, and never carries its value, which is a User's addressed post
func TestObject_ErrorsCarryNoValue(t *testing.T) {

	// O13: these errors attached the whole Object, value included, and the mongo reporter
	// stores every detail in ErrorLog
	service := Object{host: "https://example.com"}
	session := brokenSession{}
	userID := primitive.NewObjectID()

	// requireNoValue checks one error for the post's content and its recipient
	requireNoValue := func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		secretcheck.RequireAbsent(t, err, "the private message body")
		secretcheck.RequireAbsent(t, err, "private-recipient")
	}

	// newObject returns an addressed post
	newObject := func() model.Object {
		object := model.NewObject()
		object.UserID = userID
		object.Value = mapof.Any{
			vocab.PropertyType:    vocab.ObjectTypeNote,
			vocab.PropertyContent: "the private message body",
			vocab.PropertyTo:      []string{"https://example.com/@private-recipient"},
		}
		return object
	}

	t.Run("a failed save", func(t *testing.T) {
		object := newObject()
		requireNoValue(t, service.Save(session, &object, "test"))
	})

	t.Run("a failed delete", func(t *testing.T) {
		object := newObject()
		requireNoValue(t, service.Delete(session, &object, "test"))
	})

	t.Run("a failed save while creating", func(t *testing.T) {

		activity := streams.NewDocument(mapof.Any{
			vocab.PropertyType:  vocab.ActivityTypeCreate,
			vocab.PropertyActor: "https://example.com/@" + userID.Hex(),
			vocab.PropertyTo:    []string{"https://example.com/@private-recipient"},
			vocab.PropertyObject: mapof.Any{
				vocab.PropertyType:    vocab.ObjectTypeNote,
				vocab.PropertyContent: "the private message body",
				vocab.PropertyTo:      []string{"https://example.com/@private-recipient"},
			},
		})

		_, _, err := service.CreateAddressed(session, userID, activity)
		requireNoValue(t, err)
	})
}
