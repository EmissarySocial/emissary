package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestNotification_DeleteQuote pins that only the QUOTE notification for this User, Stream, and quote
// post is deleted, and that every near miss is kept
func TestNotification_DeleteQuote(t *testing.T) {

	test := newQuoteTest(t)
	service := test.service.notificationService
	userID := test.author.UserID
	streamID := primitive.NewObjectID()

	target := test.saveNotification(t, userID, streamID, model.NotificationTypeQuote, "https://remote.example/notes/1")
	otherUser := test.saveNotification(t, primitive.NewObjectID(), streamID, model.NotificationTypeQuote, "https://remote.example/notes/1")
	otherStream := test.saveNotification(t, userID, primitive.NewObjectID(), model.NotificationTypeQuote, "https://remote.example/notes/1")
	otherQuote := test.saveNotification(t, userID, streamID, model.NotificationTypeQuote, "https://remote.example/notes/2")
	otherType := test.saveNotification(t, userID, streamID, model.NotificationTypeMention, "https://remote.example/notes/1")

	require.NoError(t, service.DeleteQuote(test.session, userID, streamID, "https://remote.example/notes/1", "test"))

	require.False(t, test.hasNotification(t, target))
	require.True(t, test.hasNotification(t, otherUser), "another User's notification")
	require.True(t, test.hasNotification(t, otherStream), "a quote of another Stream")
	require.True(t, test.hasNotification(t, otherQuote), "another quote of this Stream")
	require.True(t, test.hasNotification(t, otherType), "a mention of the same post")
}

// TestNotification_DeleteQuote_EmptyURI pins that an empty quote URI deletes nothing
func TestNotification_DeleteQuote_EmptyURI(t *testing.T) {

	test := newQuoteTest(t)
	userID := test.author.UserID
	streamID := primitive.NewObjectID()

	// A QUOTE notification with no ObjectURL would match an unguarded query
	kept := test.saveNotification(t, userID, streamID, model.NotificationTypeQuote, "")

	require.NoError(t, test.service.notificationService.DeleteQuote(test.session, userID, streamID, "", "test"))
	require.True(t, test.hasNotification(t, kept))
}
