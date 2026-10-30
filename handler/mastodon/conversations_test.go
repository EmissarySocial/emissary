package mastodon

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot/object"
	"github.com/stretchr/testify/require"
)

// message builds a direct-message notification from a sender, optionally already read.
func message(actorURL string, text string, created int64, read bool) model.Notification {

	notification := model.NewNotification()
	notification.Type = model.NotificationTypeDirect
	notification.Actor = model.PersonLink{ProfileURL: actorURL, Name: "Sender", Username: "@sender@example.com"}
	notification.ObjectURL = actorURL + "/statuses/" + text
	notification.ObjectSummary = text
	notification.CreateDate = created

	if read {
		notification.ReadDate = created + 1
	}

	return notification
}

// TestGroupConversations_OneConversationPerSender confirms messages are grouped by sender, newest conversation first.
func TestGroupConversations_OneConversationPerSender(t *testing.T) {

	// Newest first, as the query returns them
	messages := []model.Notification{
		message("https://a.example.com/@amy", "amy-3", 300, true),
		message("https://b.example.com/@bo", "bo-2", 250, true),
		message("https://a.example.com/@amy", "amy-1", 100, true),
	}

	groups := groupConversations(messages, 20)

	require.Len(t, groups, 2)
	require.Equal(t, "https://a.example.com/@amy", groups[0].ActorURL)
	require.Equal(t, "amy-3", groups[0].Latest.ObjectSummary, "the latest message stands for the conversation")
	require.Equal(t, "https://b.example.com/@bo", groups[1].ActorURL)
	require.False(t, groups[0].Unread)
}

// TestGroupConversations_UnreadAndLimits covers an older unread message, the page limit, and a sender with no address.
func TestGroupConversations_UnreadAndLimits(t *testing.T) {

	messages := []model.Notification{
		message("https://a.example.com/@amy", "amy-3", 300, true),
		message("https://b.example.com/@bo", "bo-2", 250, true),
		message("https://a.example.com/@amy", "amy-1", 100, false),
		message("https://c.example.com/@cy", "cy-0", 50, true),
		message("", "nobody", 40, true),
	}

	groups := groupConversations(messages, 20)

	require.Len(t, groups, 3, "a sender with no address cannot be shown")
	require.True(t, groups[0].Unread, "an older unread message makes the whole conversation unread")
	require.False(t, groups[1].Unread)

	limited := groupConversations(messages, 2)

	require.Len(t, limited, 2)
	require.True(t, limited[0].Unread, "messages beyond the limit from a conversation already shown still count")
	require.Empty(t, groupConversations(nil, 20))
}

// TestConversationID_RoundTrips confirms an ID returns the profile address it was made from, and rejects anything else.
func TestConversationID_RoundTrips(t *testing.T) {

	id := encodeConversationID("https://mastodon.social/ap/users/117111409165181205")
	require.Equal(t, "c_", id[:2])

	address, ok := decodeConversationID(id)
	require.True(t, ok)
	require.Equal(t, "https://mastodon.social/ap/users/117111409165181205", address)

	for _, bad := range []string{"", "c_", "u_aHR0cHM6Ly9leGFtcGxl", "c_!!not-base64!!", "6aa2785d2d042f4e8b90227e"} {
		_, ok := decodeConversationID(bad)
		require.False(t, ok, bad)
	}
}

// TestDirectMessageStatus_UsesTheSavedText confirms the stand-in status is direct, dated, and safe to display.
func TestDirectMessageStatus_UsesTheSavedText(t *testing.T) {

	notification := message("https://a.example.com/@amy", "<b>hi</b> & bye", 1790000000000, false)
	account := object.Account{ID: "u_abc", Acct: "amy@a.example.com"}

	status := directMessageStatus(notification, account)

	require.Equal(t, "direct", status.Visibility)
	require.Equal(t, model.EncodeRemoteStatusID(notification.ObjectURL), status.ID)
	require.Equal(t, "<p>&lt;b&gt;hi&lt;/b&gt; &amp; bye</p>", status.Content, "the saved text is plain text, so it is escaped")
	require.Equal(t, "<b>hi</b> & bye", status.Text)
	require.Equal(t, account, status.Account)
	require.Equal(t, "2026-09-21T14:13:20.000Z", status.CreatedAt)
}

// sent builds a direct message the User sent to a recipient
func sent(recipientURL string, text string, created int64) sentMessage {
	return sentMessage{RecipientURL: recipientURL, Date: created, Note: mapof.Any{vocab.PropertyContent: text}}
}

// TestAddSentMessages_NewerSentMessageLeads confirms a reply the User sent after the last received
// message stands for the conversation, and an older one does not
func TestAddSentMessages_NewerSentMessageLeads(t *testing.T) {

	groups := groupConversations([]model.Notification{
		message("https://a.example.com/@amy", "amy-2", 200, true),
		message("https://b.example.com/@bo", "bo-1", 100, true),
	}, 20)

	groups = addSentMessages(groups, []sentMessage{
		sent("https://b.example.com/@bo", "to-bo-3", 300),
		sent("https://a.example.com/@amy", "to-amy-0", 50),
	}, 20)

	require.Len(t, groups, 2)
	require.Equal(t, "https://b.example.com/@bo", groups[0].ActorURL, "the conversation with the newest message comes first")
	require.NotNil(t, groups[0].Sent)
	require.Equal(t, "to-bo-3", groups[0].Sent.Note[vocab.PropertyContent])
	require.Equal(t, "https://a.example.com/@amy", groups[1].ActorURL)
	require.Nil(t, groups[1].Sent, "an older sent message does not replace the latest received one")
}

// TestAddSentMessages_OnlySent confirms a person the User has only messaged gets a conversation,
// within the page limit
func TestAddSentMessages_OnlySent(t *testing.T) {

	groups := addSentMessages([]conversationGroup{}, []sentMessage{
		sent("https://c.example.com/@cy", "to-cy-2", 200),
		sent("https://c.example.com/@cy", "to-cy-1", 100),
		sent("https://d.example.com/@di", "to-di-1", 150),
	}, 20)

	require.Len(t, groups, 2)
	require.Equal(t, "https://c.example.com/@cy", groups[0].ActorURL)
	require.Equal(t, "to-cy-2", groups[0].Sent.Note[vocab.PropertyContent], "the newest sent message stands for the conversation")
	require.False(t, groups[0].Unread, "a sent message is never unread")
	require.Equal(t, "https://d.example.com/@di", groups[1].ActorURL)

	limited := addSentMessages([]conversationGroup{}, []sentMessage{sent("https://c.example.com/@cy", "x", 2), sent("https://d.example.com/@di", "y", 1)}, 1)
	require.Len(t, limited, 1)
}

// TestSentTo keeps only the messages sent to the named person
func TestSentTo(t *testing.T) {

	sent := []sentMessage{
		{RecipientURL: "https://a.example/alice", Date: 3},
		{RecipientURL: "https://b.example/bob", Date: 2},
		{RecipientURL: "https://a.example/alice", Date: 1},
	}

	result := sentTo(sent, "https://a.example/alice")
	require.Len(t, result, 2)
	require.Equal(t, int64(3), result[0].Date)
	require.Equal(t, int64(1), result[1].Date)

	require.Empty(t, sentTo(sent, "https://c.example/carol"))
}
