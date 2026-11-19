package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// saveUnread saves one unread Notification of the given type for a User
func saveUnread(t *testing.T, session data.Session, userID primitive.ObjectID, notificationType string) model.Notification {

	t.Helper()

	notification := model.NewNotification()
	notification.UserID = userID
	notification.Type = notificationType
	notification.ActivityID = "https://remote.example/activities/" + notification.NotificationID.Hex()
	require.NoError(t, session.Collection("Notification").Save(&notification, "test setup"))

	return notification
}

// readDateOf returns the stored ReadDate of a Notification
func readDateOf(t *testing.T, session data.Session, notification model.Notification) int64 {

	t.Helper()

	service := Notification{}
	loaded := model.NewNotification()
	require.NoError(t, service.LoadByID(session, notification.UserID, notification.NotificationID, &loaded))

	return loaded.ReadDate
}

// TestNotification_MarkRead pins that one row is marked, and every other row is left unread
func TestNotification_MarkRead(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service := Notification{}
	userID := primitive.NewObjectID()

	clicked := saveUnread(t, session, userID, model.NotificationTypeLike)
	sameType := saveUnread(t, session, userID, model.NotificationTypeLike)
	otherType := saveUnread(t, session, userID, model.NotificationTypeFollow)

	require.NoError(t, service.MarkRead(session, userID, clicked.NotificationID, 1234))

	require.Equal(t, int64(1234), readDateOf(t, session, clicked))
	require.Equal(t, sameType.ReadDate, readDateOf(t, session, sameType), "a second row of the same type stays unread")
	require.Equal(t, otherType.ReadDate, readDateOf(t, session, otherType))

	unread, err := service.CountUnread(session, userID)
	require.NoError(t, err)
	require.Equal(t, int64(2), unread)
}

// TestNotification_MarkRead_OtherUser pins that nobody can mark another User's Notification
func TestNotification_MarkRead_OtherUser(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service := Notification{}

	notification := saveUnread(t, session, primitive.NewObjectID(), model.NotificationTypeMention)

	// The update matches nothing, which is not an error
	require.NoError(t, service.MarkRead(session, primitive.NewObjectID(), notification.NotificationID, 1234))
	require.Equal(t, notification.ReadDate, readDateOf(t, session, notification))
}

// TestNotification_MarkRead_KeepsFirstReadDate pins that a second click does not move the read date
func TestNotification_MarkRead_KeepsFirstReadDate(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service := Notification{}
	userID := primitive.NewObjectID()

	notification := saveUnread(t, session, userID, model.NotificationTypeReply)

	require.NoError(t, service.MarkRead(session, userID, notification.NotificationID, 1000))
	require.NoError(t, service.MarkRead(session, userID, notification.NotificationID, 2000))
	require.Equal(t, int64(1000), readDateOf(t, session, notification))
}

// TestNotification_MarkRead_Missing pins that a purged or unknown Notification is not an error
func TestNotification_MarkRead_Missing(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service := Notification{}

	require.NoError(t, service.MarkRead(session, primitive.NewObjectID(), primitive.NewObjectID(), 1234))
}

// TestNotification_MarkAllRead_ByTab pins that marking a tab leaves the other types unread
func TestNotification_MarkAllRead_ByTab(t *testing.T) {

	_, session := newReplicaSetSession(t)
	service := Notification{}
	userID := primitive.NewObjectID()

	like := saveUnread(t, session, userID, model.NotificationTypeLike)
	quote := saveUnread(t, session, userID, model.NotificationTypeQuote)

	require.NoError(t, service.MarkAllRead(session, userID, 1234, model.NotificationTypeQuote))
	require.Equal(t, int64(1234), readDateOf(t, session, quote))
	require.Equal(t, like.ReadDate, readDateOf(t, session, like))

	// No types means every type, which is what the All tab sends
	require.NoError(t, service.MarkAllRead(session, userID, 5678))
	require.Equal(t, int64(5678), readDateOf(t, session, like))
	require.Equal(t, int64(1234), readDateOf(t, session, quote), "a row already read keeps its date")
}
