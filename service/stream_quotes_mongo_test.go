package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// quoteTest holds a Stream service wired for the quote methods against a real MongoDB
type quoteTest struct {
	service Stream
	session data.Session
	tasks   *postcommit.Tasks
	author  model.User
}

// spooledSession wraps a real session, spooling its post-commit tasks so a test can read them
type spooledSession struct {
	data.Session
	tasks *postcommit.Tasks
}

// Context carries the post-commit spool
func (s spooledSession) Context() context.Context {
	return postcommit.WithContext(s.Session.Context(), s.tasks)
}

// newStream saves and returns a public Stream by this test's author, open to replies and quotes
func (test quoteTest) newStream(t *testing.T) model.Stream {

	t.Helper()

	stream := model.NewStream()
	stream.URL = "https://local.example/" + stream.StreamID.Hex()
	stream.DefaultAllow = model.Permissions{model.MagicGroupIDAnonymous}
	stream.AttributedTo = model.PersonLink{UserID: test.author.UserID, ProfileURL: test.author.ProfileURL}
	require.NoError(t, test.session.Collection("Stream").Save(&stream, "test setup"))

	return stream
}

// saveNotification saves one Notification that tells the User about the quote post, for the provided Stream
func (test quoteTest) saveNotification(t *testing.T, userID primitive.ObjectID, streamID primitive.ObjectID, notificationType string, objectURL string) model.Notification {

	t.Helper()

	notification := model.NewNotification()
	notification.UserID = userID
	notification.Type = notificationType
	notification.ActivityID = "https://remote.example/activities/" + notification.NotificationID.Hex()
	notification.ObjectURL = objectURL
	notification.StreamID = streamID
	notification.Actor = model.PersonLink{ProfileURL: "https://remote.example/users/bob"}
	require.NoError(t, test.session.Collection("Notification").Save(&notification, "test setup"))

	return notification
}

// hasNotification returns TRUE if the Notification is still stored for its User
func (test quoteTest) hasNotification(t *testing.T, notification model.Notification) bool {

	t.Helper()

	loaded := model.NewNotification()
	err := test.service.notificationService.LoadByID(test.session, notification.UserID, notification.NotificationID, &loaded)

	if derp.IsNotFound(err) {
		return false
	}

	require.NoError(t, err)
	return true
}

// TestStream_ApproveQuote_OneItemPerQuote pins that each quote post is one CollectionItem in the Stream's
// Quotes collection, and that approving the same quote again returns the same item and QuoteAuthorization
func TestStream_ApproveQuote_OneItemPerQuote(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	first, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)
	require.Equal(t, model.CollectionTypeQuotes, first.CollectionType)
	require.Equal(t, "https://remote.example/notes/1", first.URI)
	require.Equal(t, stream.StreamID, first.ParentID)
	require.Equal(t, test.author.UserID, first.UserID)

	again, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)
	require.Equal(t, first.CollectionItemID, again.CollectionItemID, "a repeated approval reuses its item")

	second, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/2")
	require.NoError(t, err)
	require.NotEqual(t, first.CollectionItemID, second.CollectionItemID)

	collection := model.NewCollection()
	require.NoError(t, test.service.collectionService.LoadByType(test.session, stream.StreamID, model.CollectionTypeQuotes, &collection))
	require.Equal(t, 2, collection.TotalItems)

	// Approving sends nothing on its own; the inbox handler sends the Accept
	require.Empty(t, test.tasks.Drain())
}

// TestStream_ApproveQuote_Refusals pins the guards that run before any database work
func TestStream_ApproveQuote_Refusals(t *testing.T) {

	// An in-memory store proves that a refusal writes nothing
	store := &itemStore{}
	collectionItemService, session := newItemService(store)
	service := Stream{
		collectionService:     &Collection{collectionItemService: collectionItemService},
		collectionItemService: collectionItemService,
	}

	stream := model.NewStream()
	stream.DefaultAllow = model.Permissions{model.MagicGroupIDAnonymous}

	_, err := service.ApproveQuote(session, &stream, "")
	require.True(t, derp.IsBadRequest(err), "an approval must name its quote post")

	stream.ReplyPolicy = model.ReplyPolicyNobody
	_, err = service.ApproveQuote(session, &stream, "https://remote.example/notes/1")
	require.True(t, derp.IsForbidden(err), "a closed Stream approves no quotes")

	stream.ReplyPolicy = ""
	stream.DefaultAllow = model.Permissions{model.MagicGroupIDAuthenticated}
	_, err = service.ApproveQuote(session, &stream, "https://remote.example/notes/1")
	require.True(t, derp.IsForbidden(err), "a private Stream approves no quotes")

	require.Empty(t, store.records)
}

// TestStream_LoadQuote_FollowsCurrentPolicy pins that a QuoteAuthorization is served only while the Stream
// accepts quotes, and that closing the Stream hides it without deleting it
func TestStream_LoadQuote_FollowsCurrentPolicy(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	approved, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	loaded := model.NewCollectionItem()
	require.NoError(t, test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded))
	require.Equal(t, approved.CollectionItemID, loaded.CollectionItemID)
	require.Equal(t, "https://remote.example/notes/1", loaded.URI)

	// Closed: NotFound, although the item is still stored
	stream.ReplyPolicy = model.ReplyPolicyNobody
	require.True(t, derp.IsNotFound(test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded)))

	// Reopened: the same QuoteAuthorization is served again
	stream.ReplyPolicy = ""
	require.NoError(t, test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded))
}

// TestStream_LoadQuote_OnlyThisStreamsQuotes pins that the route serves no item from another Stream,
// and no item from another of this Stream's collections
func TestStream_LoadQuote_OnlyThisStreamsQuotes(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)
	other := test.newStream(t)

	approved, err := test.service.ApproveQuote(test.session, &other, "https://remote.example/notes/1")
	require.NoError(t, err)

	loaded := model.NewCollectionItem()
	require.True(t, derp.IsNotFound(test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded)), "another Stream's quote")

	// A Like on this Stream is a CollectionItem too, but not a QuoteAuthorization
	likes, err := test.service.collectionService.LoadOrCreateByStream(test.session, &stream, model.CollectionTypeLikes)
	require.NoError(t, err)
	require.NoError(t, test.service.collectionService.AddItem(test.session, &likes, "https://remote.example/likes/1", stream.URL))

	like := model.NewCollectionItem()
	require.NoError(t, test.service.collectionItemService.LoadByURI(test.session, likes.CollectionID, "https://remote.example/likes/1", &like))
	require.True(t, derp.IsNotFound(test.service.LoadQuote(test.session, &stream, like.CollectionItemID, &loaded)), "a Like")

	// An ID that names nothing at all
	require.True(t, derp.IsNotFound(test.service.LoadQuote(test.session, &stream, primitive.NewObjectID(), &loaded)), "an unknown ID")
}

// TestStream_RejectQuote pins that rejecting a quote deletes its item and its QUOTE notification, so its
// QuoteAuthorization stops resolving, and queues one task that carries everything needed to send its Delete
func TestStream_RejectQuote(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	approved, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	authorizationURL := stream.ActivityPubQuoteAuthorizationURL(approved.CollectionItemID)
	quoted := test.saveNotification(t, test.author.UserID, stream.StreamID, model.NotificationTypeQuote, "https://remote.example/notes/1")
	mentioned := test.saveNotification(t, test.author.UserID, stream.StreamID, model.NotificationTypeMention, "https://remote.example/notes/1")

	require.NoError(t, test.service.RejectQuote(test.session, &stream, "https://remote.example/notes/1"))

	require.False(t, test.hasNotification(t, quoted), "the QUOTE notification is removed")
	require.True(t, test.hasNotification(t, mentioned), "a mention of the same post is kept")

	// The QuoteAuthorization no longer resolves
	loaded := model.NewCollectionItem()
	require.True(t, derp.IsNotFound(test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded)))

	collection := model.NewCollection()
	require.NoError(t, test.service.collectionService.LoadByType(test.session, stream.StreamID, model.CollectionTypeQuotes, &collection))
	require.Equal(t, 0, collection.TotalItems)

	// One task is queued.  It names the quote post, from which the queue finds the quoting actor
	tasks := test.tasks.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, TaskSendDeleteQuoteAuthorization, tasks[0].Name)
	require.Equal(t, mapof.Any{
		"hostname":      "https://local.example",
		"actor":         "https://local.example/@author",
		"followers":     "https://local.example/@author/pub/followers",
		"authorization": authorizationURL,
		"attributedTo":  "https://local.example/@author",
		"quote":         "https://remote.example/notes/1",
		"target":        stream.URL,
	}, tasks[0].Arguments)

	// Rejecting it again does nothing, and sends nothing
	require.NoError(t, test.service.RejectQuote(test.session, &stream, "https://remote.example/notes/1"))
	require.Empty(t, test.tasks.Drain())

	// A rejected quote can be approved again, with a new QuoteAuthorization
	reapproved, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)
	require.NotEqual(t, approved.CollectionItemID, reapproved.CollectionItemID)
}

// TestStream_RejectQuote_NeverQuoted pins that rejecting a quote of a Stream that has none does nothing
func TestStream_RejectQuote_NeverQuoted(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	require.NoError(t, test.service.RejectQuote(test.session, &stream, "https://remote.example/notes/1"))
	require.Empty(t, test.tasks.Drain())
}

// TestStream_RejectQuote_OtherQuotesRemain pins that rejecting one quote leaves the Stream's other quotes approved
func TestStream_RejectQuote_OtherQuotesRemain(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	_, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	kept, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/2")
	require.NoError(t, err)

	require.NoError(t, test.service.RejectQuote(test.session, &stream, "https://remote.example/notes/1"))

	loaded := model.NewCollectionItem()
	require.NoError(t, test.service.LoadQuote(test.session, &stream, kept.CollectionItemID, &loaded))
	require.Equal(t, "https://remote.example/notes/2", loaded.URI)
}

// TestStream_RejectQuote_WhileClosed pins that the author can reject a quote while the Stream refuses
// quotes, so the quoting server is told without waiting to re-fetch the QuoteAuthorization
func TestStream_RejectQuote_WhileClosed(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	_, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	stream.ReplyPolicy = model.ReplyPolicyNobody
	require.NoError(t, test.service.RejectQuote(test.session, &stream, "https://remote.example/notes/1"))

	tasks := test.tasks.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, "https://remote.example/notes/1", tasks[0].Arguments.GetString("quote"))

	collection := model.NewCollection()
	require.NoError(t, test.service.collectionService.LoadByType(test.session, stream.StreamID, model.CollectionTypeQuotes, &collection))
	require.Equal(t, 0, collection.TotalItems)
}

// TestStream_RejectQuote_DeletedStream pins that a quote of a deleted Stream loses its notification
// and sends nothing, because a deleted Stream serves no QuoteAuthorizations
func TestStream_RejectQuote_DeletedStream(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	_, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	quoted := test.saveNotification(t, test.author.UserID, stream.StreamID, model.NotificationTypeQuote, "https://remote.example/notes/1")
	require.NoError(t, test.session.Collection("Stream").Delete(&stream, "test setup"))

	require.NoError(t, test.service.RejectQuote(test.session, &stream, "https://remote.example/notes/1"))

	require.Empty(t, test.tasks.Drain())
	require.False(t, test.hasNotification(t, quoted))
}

// TestCollectionItem_ObjectDelete_RevokesQuote pins that the generic delete, which the `delete` build
// step calls, revokes a quote in full: the item, the count, the notification, and the federated Delete
func TestCollectionItem_ObjectDelete_RevokesQuote(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	approved, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	quoted := test.saveNotification(t, test.author.UserID, stream.StreamID, model.NotificationTypeQuote, "https://remote.example/notes/1")

	require.NoError(t, test.service.collectionItemService.ObjectDelete(test.session, &approved, "test"))

	loaded := model.NewCollectionItem()
	require.True(t, derp.IsNotFound(test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded)))

	collection := model.NewCollection()
	require.NoError(t, test.service.collectionService.LoadByType(test.session, stream.StreamID, model.CollectionTypeQuotes, &collection))
	require.Equal(t, 0, collection.TotalItems)

	tasks := test.tasks.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, "https://remote.example/notes/1", tasks[0].Arguments.GetString("quote"))
	require.False(t, test.hasNotification(t, quoted))
}

// TestCollectionItem_Delete_MissingQuote pins that deleting a quote that is already gone, as the
// with-collection-item step does on a repeated POST, succeeds and sends nothing
func TestCollectionItem_Delete_MissingQuote(t *testing.T) {

	test := newQuoteTest(t)

	// The step names a missing item by its ID, its owner, and its type alone
	missing := model.NewCollectionItem()
	missing.UserID = test.author.UserID
	missing.CollectionType = model.CollectionTypeQuotes

	require.NoError(t, test.service.collectionItemService.Delete(test.session, &missing, "test"))
	require.Empty(t, test.tasks.Drain())
}

// TestStream_LoadQuoteByURI pins that a quote is found from its Stream and the URI of the quote post,
// which is what a QUOTE notification stores, and that every near miss is NotFound
func TestStream_LoadQuoteByURI(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)
	other := test.newStream(t)

	approved, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	// A Like of the same Stream from the same URI is not a quote
	likes, err := test.service.collectionService.LoadOrCreateByStream(test.session, &other, model.CollectionTypeLikes)
	require.NoError(t, err)
	require.NoError(t, test.service.collectionService.AddItem(test.session, &likes, "https://remote.example/notes/1", other.URL))

	found := model.NewCollectionItem()
	require.NoError(t, test.service.LoadQuoteByURI(test.session, stream.StreamID, "https://remote.example/notes/1", &found))
	require.Equal(t, approved.CollectionItemID, found.CollectionItemID)

	// It is found while the Stream is closed too, so a closed Stream's quotes can still be rejected
	stream.ReplyPolicy = model.ReplyPolicyNobody
	require.NoError(t, test.service.LoadQuoteByURI(test.session, stream.StreamID, "https://remote.example/notes/1", &found))

	miss := model.NewCollectionItem()
	require.True(t, derp.IsNotFound(test.service.LoadQuoteByURI(test.session, stream.StreamID, "https://remote.example/notes/2", &miss)), "a post that never quoted")
	require.True(t, derp.IsNotFound(test.service.LoadQuoteByURI(test.session, other.StreamID, "https://remote.example/notes/1", &miss)), "a Stream that holds only a Like from that URI")
	require.True(t, derp.IsNotFound(test.service.LoadQuoteByURI(test.session, primitive.NewObjectID(), "https://remote.example/notes/1", &miss)), "a Stream that does not exist")
	require.True(t, derp.IsNotFound(test.service.LoadQuoteByURI(test.session, stream.StreamID, "", &miss)), "an empty URI")

	// Rejected, it is gone
	require.NoError(t, test.service.RejectQuote(test.session, &stream, "https://remote.example/notes/1"))
	require.True(t, derp.IsNotFound(test.service.LoadQuoteByURI(test.session, stream.StreamID, "https://remote.example/notes/1", &miss)), "a rejected quote")
}

// TestCollectionItem_Delete_OnlyQuotesFederate pins that deleting an item of any other collection
// type sends nothing, and still refreshes the count
func TestCollectionItem_Delete_OnlyQuotesFederate(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	likes, err := test.service.collectionService.LoadOrCreateByStream(test.session, &stream, model.CollectionTypeLikes)
	require.NoError(t, err)
	require.NoError(t, test.service.collectionService.AddItem(test.session, &likes, "https://remote.example/likes/1", stream.URL))

	like := model.NewCollectionItem()
	require.NoError(t, test.service.collectionItemService.LoadByURI(test.session, likes.CollectionID, "https://remote.example/likes/1", &like))
	require.NoError(t, test.service.collectionItemService.Delete(test.session, &like, "test"))

	require.Empty(t, test.tasks.Drain())
	require.NoError(t, test.service.collectionService.LoadByType(test.session, stream.StreamID, model.CollectionTypeLikes, &likes))
	require.Equal(t, 0, likes.TotalItems)
}

// TestCollectionItem_HardDeleteByID_RevokesQuote pins that the hard delete revokes a quote for its
// owner, and does nothing at all for any other User
func TestCollectionItem_HardDeleteByID_RevokesQuote(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)
	service := test.service.collectionItemService

	approved, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	// Another User cannot delete the item
	require.NoError(t, service.HardDeleteByID(test.session, primitive.NewObjectID(), approved.CollectionItemID))
	require.Empty(t, test.tasks.Drain())

	loaded := model.NewCollectionItem()
	require.NoError(t, test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded))

	// Its owner can, and the quoting actor is told
	require.NoError(t, service.HardDeleteByID(test.session, test.author.UserID, approved.CollectionItemID))
	require.Len(t, test.tasks.Drain(), 1)
	require.True(t, derp.IsNotFound(test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded)))
}

// TestCollectionItem_HardDeleteByURI_RevokesQuote pins that deleting by URI revokes the quote too
func TestCollectionItem_HardDeleteByURI_RevokesQuote(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	approved, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	require.NoError(t, test.service.collectionItemService.HardDeleteByURI(test.session, "https://remote.example/notes/1"))
	require.Len(t, test.tasks.Drain(), 1)

	loaded := model.NewCollectionItem()
	require.True(t, derp.IsNotFound(test.service.LoadQuote(test.session, &stream, approved.CollectionItemID, &loaded)))
}

// TestCollection_Delete_RevokesEveryQuote pins that deleting a Quotes collection revokes each quote in it
func TestCollection_Delete_RevokesEveryQuote(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)

	_, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	_, err = test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/2")
	require.NoError(t, err)

	collection := model.NewCollection()
	require.NoError(t, test.service.collectionService.LoadByType(test.session, stream.StreamID, model.CollectionTypeQuotes, &collection))
	require.NoError(t, test.service.collectionService.Delete(test.session, &collection, "test"))

	require.Len(t, test.tasks.Drain(), 2)
}

// TestCollectionItem_LoadByUserID pins that an item loads only for the User who owns it
func TestCollectionItem_LoadByUserID(t *testing.T) {

	test := newQuoteTest(t)
	stream := test.newStream(t)
	service := test.service.collectionItemService

	approved, err := test.service.ApproveQuote(test.session, &stream, "https://remote.example/notes/1")
	require.NoError(t, err)

	loaded := model.NewCollectionItem()
	require.NoError(t, service.LoadByUserID(test.session, test.author.UserID, approved.CollectionItemID, &loaded))
	require.Equal(t, approved.URI, loaded.URI)

	other := model.NewCollectionItem()
	require.True(t, derp.IsNotFound(service.LoadByUserID(test.session, primitive.NewObjectID(), approved.CollectionItemID, &other)))
}

// newQuoteTest returns a Stream service wired with every service the quote methods call, and a saved author
func newQuoteTest(t *testing.T) quoteTest {

	t.Helper()

	_, session := newReplicaSetSession(t)
	tasks := postcommit.NewTasks()
	spooled := spooledSession{Session: session, tasks: tasks}

	collectionItemService := &CollectionItem{}
	collectionService := &Collection{collectionItemService: collectionItemService, locatorService: &Locator{host: "https://local.example"}}

	service := Stream{
		collectionService:     collectionService,
		collectionItemService: collectionItemService,
		notificationService:   &Notification{},
		outboxService:         &Outbox{host: "https://local.example"},
		userService:           &User{},
	}

	// Deleting a CollectionItem reaches back up to its Collection and its Stream
	collectionItemService.collectionService = collectionService
	collectionItemService.streamService = &service

	author := model.NewUser()
	author.ProfileURL = "https://local.example/@author"
	require.NoError(t, spooled.Collection("User").Save(&author, "test setup"))

	return quoteTest{service: service, session: spooled, tasks: tasks, author: author}
}
