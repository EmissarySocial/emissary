package mastodon

import (
	"encoding/base64"
	"html"
	"strings"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

const (
	conversationIDPrefix = "c_" // Marks a conversation ID, which packs in the other person's profile address
	conversationScan     = 200  // Most direct messages read to build the list
	conversationWorkers  = 8    // Conversations built at the same time
)

// conversationGroup is one person's direct messages to the signed-in User.
type conversationGroup struct {
	ActorURL string
	Latest   model.Notification
	Unread   bool
}

// encodeConversationID turns the other person's profile address into a conversation ID.
func encodeConversationID(actorURL string) string {
	return conversationIDPrefix + base64.RawURLEncoding.EncodeToString([]byte(actorURL))
}

// decodeConversationID reverses encodeConversationID, reporting false for anything else.
func decodeConversationID(id string) (string, bool) {

	encoded, found := strings.CutPrefix(id, conversationIDPrefix)

	if !found {
		return "", false
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)

	if err != nil || len(raw) == 0 {
		return "", false
	}

	return string(raw), true
}

// groupConversations gathers direct messages (newest first) into one conversation per sender, ordered by
// each conversation's latest message and stopping at limit. A conversation is unread if any of its messages is.
func groupConversations(messages []model.Notification, limit int) []conversationGroup {

	result := []conversationGroup{}
	index := map[string]int{}

	for _, message := range messages {

		actorURL := message.Actor.ProfileURL

		if actorURL == "" {
			continue
		}

		position, found := index[actorURL]

		if !found {

			if len(result) >= limit {
				continue
			}

			index[actorURL] = len(result)
			result = append(result, conversationGroup{ActorURL: actorURL, Latest: message})
			position = len(result) - 1
		}

		if message.NotRead() {
			result[position].Unread = true
		}
	}

	return result
}

// directMessageStatus builds a Status from what a direct-message notification saved, for when the
// message itself cannot be fetched from the sender's server.
func directMessageStatus(notification model.Notification, account object.Account) object.Status {

	return object.Status{
		ID:         model.EncodeRemoteStatusID(notification.ObjectURL),
		URI:        notification.ObjectURL,
		URL:        notification.ObjectURL,
		CreatedAt:  model.MastodonDate(time.UnixMilli(notification.CreateDate)),
		Visibility: "direct",
		Account:    account,
		Content:    "<p>" + html.EscapeString(notification.ObjectSummary) + "</p>",
		Text:       notification.ObjectSummary,
	}
}

// conversationToToot builds the Conversation for one person's messages: the sender as the participant,
// and their latest message as the last status.
func conversationToToot(factory *service.Factory, session data.Session, auth model.Authorization, group conversationGroup, accounts *accountMemo) object.Conversation {

	client := factory.ActivityStream().UserClient(auth.UserID)

	account, _ := accounts.get(group.ActorURL, func() (object.Account, bool) {

		if loaded, found := loadAccount(client, factory, session, group.ActorURL); found {
			return loaded, true
		}

		return group.Latest.Actor.Toot(), true
	})

	status, err := statusForPostURL(factory, session, auth, group.Latest.ObjectURL, "handler.mastodon_conversationToToot")

	if err != nil {
		status = directMessageStatus(group.Latest, account)
	}

	status.Visibility = "direct"

	return object.Conversation{
		ID:         encodeConversationID(group.ActorURL),
		Unread:     group.Unread,
		Accounts:   []object.Account{account},
		LastStatus: status,
	}
}

// https://docs.joinmastodon.org/methods/conversations/

// GetConversations implements the Mastodon "get conversations" endpoint: one conversation per person who has sent
// the User a direct message, newest first, as one page. Encrypted messages cannot be shown and are left out.
func GetConversations(serverFactory *server.Factory) func(model.Authorization, txn.GetConversations) ([]object.Conversation, toot.PageInfo, error) {

	const location = "handler.mastodon.GetConversations"

	return func(auth model.Authorization, t txn.GetConversations) ([]object.Conversation, toot.PageInfo, error) {

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return []object.Conversation{}, toot.PageInfo{}, err
		}

		defer cancel()

		// Read the User's latest direct messages
		criteria := exp.Equal("type", model.NotificationTypeDirect).AndNotEqual("subtype", model.NotificationSubtypeMLS)
		messages, err := factory.Notification().QueryByUserID(session, auth.UserID, criteria, option.SortDesc("createDate"), option.MaxRows(conversationScan))

		if err != nil {
			return []object.Conversation{}, toot.PageInfo{}, derp.Wrap(err, location, "Querying direct messages")
		}

		// Group them into one conversation per sender
		groups := groupConversations(messages, int(pageLimit(t.Limit)))
		result := make([]object.Conversation, len(groups))
		accounts := newAccountMemo()

		// Build the conversations in parallel, keeping their order

		var waitGroup sync.WaitGroup
		slots := make(chan struct{}, conversationWorkers)

		for index := range groups {

			waitGroup.Add(1)
			slots <- struct{}{}

			go func(index int) {
				defer waitGroup.Done()
				defer func() { <-slots }()
				result[index] = conversationToToot(factory, session, auth, groups[index], accounts)
			}(index)
		}

		waitGroup.Wait()

		return result, toot.PageInfo{}, nil
	}
}

// DeleteConversation implements the Mastodon "delete conversation" endpoint as a no-op
func DeleteConversation(serverFactory *server.Factory) func(model.Authorization, txn.DeleteConversation) (struct{}, error) {

	return func(auth model.Authorization, t txn.DeleteConversation) (struct{}, error) {
		return struct{}{}, nil
	}
}

// PostConversationRead implements the Mastodon "mark conversation read" endpoint as a no-op
func PostConversationRead(serverFactory *server.Factory) func(model.Authorization, txn.PostConversationRead) (struct{}, error) {

	return func(auth model.Authorization, t txn.PostConversationRead) (struct{}, error) {
		return struct{}{}, nil
	}
}
