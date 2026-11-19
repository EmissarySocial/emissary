package mastodon

import (
	"cmp"
	"encoding/base64"
	"html"
	"slices"
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
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	conversationIDPrefix = "c_" // Marks a conversation ID, which packs in the other person's profile address
	conversationScan     = 200  // Most direct messages read to build the list
	conversationWorkers  = 8    // Conversations built at the same time
)

// conversationGroup is the signed-in User's direct messages with one person: the latest one they
// received from them, and the latest one they sent to them when that is newer.
type conversationGroup struct {
	ActorURL string
	Latest   model.Notification
	Sent     *sentMessage
	Unread   bool
}

// sentMessage is a direct message the signed-in User sent to one person
type sentMessage struct {
	RecipientURL string
	Date         int64
	Note         mapof.Any
}

// latestDate returns when the conversation's newest message, received or sent, was created
func (group conversationGroup) latestDate() int64 {

	if group.Sent != nil && group.Sent.Date > group.Latest.CreateDate {
		return group.Sent.Date
	}

	return group.Latest.CreateDate
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

// addSentMessages merges the User's sent direct messages (newest first) into the received conversations,
// adding one for a person only messaged while there is room, and orders them newest first
func addSentMessages(groups []conversationGroup, sent []sentMessage, limit int) []conversationGroup {

	index := make(map[string]int, len(groups))

	for position, group := range groups {
		index[group.ActorURL] = position
	}

	for _, message := range sent {

		position, found := index[message.RecipientURL]

		if !found {

			if len(groups) >= limit {
				continue
			}

			index[message.RecipientURL] = len(groups)
			groups = append(groups, conversationGroup{ActorURL: message.RecipientURL, Sent: &message})
			continue
		}

		// Only the newest sent message can stand for the conversation
		if groups[position].Sent == nil && message.Date > groups[position].Latest.CreateDate {
			groups[position].Sent = &message
		}
	}

	slices.SortStableFunc(groups, func(a conversationGroup, b conversationGroup) int {
		return cmp.Compare(b.latestDate(), a.latestDate())
	})

	return groups
}

// sentDirectMessages returns the direct messages a User has sent, newest first, once per recipient.
func sentDirectMessages(factory *service.Factory, session data.Session, userID primitive.ObjectID) ([]sentMessage, error) {

	criteria := exp.Equal("userId", userID).AndEqual("value.type", vocab.ObjectTypeNote)
	objects, err := factory.Object().Range(session, criteria, option.SortDesc("createDate"), option.MaxRows(conversationScan))

	if err != nil {
		return nil, derp.Wrap(err, "handler.mastodon.sentDirectMessages", "Querying sent messages")
	}

	result := make([]sentMessage, 0)

	for object := range objects {
		for _, recipient := range object.Permissions {
			result = append(result, sentMessage{RecipientURL: recipient, Date: object.CreateDate, Note: object.Value})
		}
	}

	return result, nil
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
func conversationToToot(factory *service.Factory, session data.Session, auth model.Authorization, group conversationGroup, self object.Account, accounts *accountMemo) object.Conversation {

	client := factory.ActivityStream().UserClient(auth.UserID)

	account, _ := accounts.get(group.ActorURL, func() (object.Account, bool) {

		if loaded, found := loadAccount(client, factory, session, group.ActorURL); found {
			return loaded, true
		}

		// A person the User has only sent messages to has no saved notification to describe them
		if group.Latest.Actor.ProfileURL == "" {
			return model.RemoteActorAccount(group.ActorURL, "", "", time.Time{}, factory.Host()), true
		}

		return group.Latest.Actor.Toot(factory.Host()), true
	})

	var status object.Status

	if group.Sent != nil && group.Sent.Date >= group.latestDate() {

		// The User's own message is saved here, so it never needs fetching
		status = documentToStatus(streams.NewDocument(group.Sent.Note), self)

	} else {

		loaded, err := statusForPostURL(factory, session, auth, group.Latest.ObjectURL, "handler.mastodon_conversationToToot")

		if err != nil {
			loaded = directMessageStatus(group.Latest, account)
		}

		status = loaded
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

// GetConversations implements the Mastodon "get conversations" endpoint: one conversation per person the User has
// exchanged direct messages with, newest first, as one page. Encrypted messages cannot be shown and are left out.
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

		// Group them into one conversation per sender, then add the messages the User sent
		limit := int(pageLimit(t.Limit))
		groups := groupConversations(messages, limit)

		sent, err := sentDirectMessages(factory, session, auth.UserID)

		if err != nil {
			return []object.Conversation{}, toot.PageInfo{}, derp.Wrap(err, location, "Reading sent messages")
		}

		groups = addSentMessages(groups, sent, limit)

		user := model.NewUser()

		if err := factory.User().LoadByID(session, auth.UserID, &user); err != nil {
			return []object.Conversation{}, toot.PageInfo{}, derp.Wrap(err, location, "Loading user")
		}

		self := user.Toot(factory.Host())
		result := make([]object.Conversation, len(groups))
		accounts := newAccountMemo()

		// Build the conversations in parallel, keeping their order

		var waitGroup sync.WaitGroup
		slots := make(chan struct{}, conversationWorkers)

		for index := range groups {

			waitGroup.Add(1)
			slots <- struct{}{}

			go func(index int) {
				defer recoverGoroutine(location)
				defer waitGroup.Done()
				defer func() { <-slots }()
				result[index] = conversationToToot(factory, session, auth, groups[index], self, accounts)
			}(index)
		}

		waitGroup.Wait()

		return result, toot.PageInfo{}, nil
	}
}

// conversationMessages returns the User's direct messages from the person a conversation ID names.
func conversationMessages(factory *service.Factory, session data.Session, auth model.Authorization, conversationID string) ([]model.Notification, error) {

	actorURL, ok := decodeConversationID(conversationID)

	if !ok {
		return nil, derp.NotFound("handler.mastodon.conversationMessages", "Unrecognized conversation", conversationID)
	}

	criteria := exp.Equal("type", model.NotificationTypeDirect).AndEqual("actor.profileUrl", actorURL)
	return factory.Notification().QueryByUserID(session, auth.UserID, criteria)
}

// DeleteConversation implements the Mastodon "remove conversation" endpoint, which removes the User's
// direct messages from that person. One that is already gone counts as done.
func DeleteConversation(serverFactory *server.Factory) func(model.Authorization, txn.DeleteConversation) (struct{}, error) {

	const location = "handler.mastodon.DeleteConversation"

	return func(auth model.Authorization, t txn.DeleteConversation) (struct{}, error) {

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return struct{}{}, err
		}

		defer cancel()

		messages, err := conversationMessages(factory, session, auth, t.ID)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Finding conversation", t.ID, derp.WithBadRequest())
		}

		for index := range messages {
			if err := factory.Notification().Delete(session, &messages[index], "Removed via Mastodon API"); err != nil {
				return struct{}{}, derp.Wrap(err, location, "Removing message", t.ID)
			}
		}

		return struct{}{}, nil
	}
}

// PostConversationRead implements the Mastodon "mark conversation read" endpoint.
func PostConversationRead(serverFactory *server.Factory) func(model.Authorization, txn.PostConversationRead) (struct{}, error) {

	const location = "handler.mastodon.PostConversationRead"

	return func(auth model.Authorization, t txn.PostConversationRead) (struct{}, error) {

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return struct{}{}, err
		}

		defer cancel()

		messages, err := conversationMessages(factory, session, auth, t.ID)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Finding conversation", t.ID, derp.WithBadRequest())
		}

		for index := range messages {

			if messages[index].IsRead() {
				continue
			}

			messages[index].MarkRead()

			if err := factory.Notification().Save(session, &messages[index], "Read via Mastodon API"); err != nil {
				return struct{}{}, derp.Wrap(err, location, "Marking message read", t.ID)
			}
		}

		return struct{}{}, nil
	}
}

// conversationForActor builds the User's conversation with one person, as GetConversations lists it
func conversationForActor(factory *service.Factory, session data.Session, auth model.Authorization, actorURL string) (object.Conversation, error) {

	const location = "handler.mastodon.conversationForActor"

	// That person's direct messages to the User (encrypted ones cannot be shown)
	criteria := exp.Equal("type", model.NotificationTypeDirect).
		AndNotEqual("subtype", model.NotificationSubtypeMLS).
		AndEqual("actor.profileUrl", actorURL)

	messages, err := factory.Notification().QueryByUserID(session, auth.UserID, criteria, option.SortDesc("createDate"), option.MaxRows(conversationScan))

	if err != nil {
		return object.Conversation{}, derp.Wrap(err, location, "Querying direct messages", actorURL)
	}

	sent, err := sentDirectMessages(factory, session, auth.UserID)

	if err != nil {
		return object.Conversation{}, derp.Wrap(err, location, "Reading sent messages")
	}

	groups := addSentMessages(groupConversations(messages, 1), sentTo(sent, actorURL), 1)

	if len(groups) == 0 {
		return object.Conversation{}, derp.NotFound(location, "No direct messages with this person", actorURL)
	}

	user := model.NewUser()

	if err := factory.User().LoadByID(session, auth.UserID, &user); err != nil {
		return object.Conversation{}, derp.Wrap(err, location, "Loading user")
	}

	return conversationToToot(factory, session, auth, groups[0], user.Toot(factory.Host()), newAccountMemo()), nil
}

// sentTo returns the sent messages addressed to one person
func sentTo(sent []sentMessage, recipientURL string) []sentMessage {

	result := make([]sentMessage, 0)

	for _, message := range sent {
		if message.RecipientURL == recipientURL {
			result = append(result, message)
		}
	}

	return result
}
