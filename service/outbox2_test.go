package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestOutbox2_Send pins the record-less send contract (PROFILE-UPDATE-FEDERATION.md D-1):
// an activity without an actor is rejected (the send consumer signs with the actor's key,
// so an actor-less activity could never deliver), while a well-formed activity is accepted.
// The mock session carries no queue and no transaction spool, so "accepted" here means the
// postcommit gate ran without error; actual delivery is covered by the federation matrix.
func TestOutbox2_Send(t *testing.T) {

	service := Outbox2{}

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	// RULE: An activity without an actor is rejected
	missingActor := mapof.Any{
		vocab.PropertyType:   vocab.ActivityTypeUpdate,
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: "https://example.com/@alice"},
	}

	require.Error(t, service.Send(session, missingActor))

	// A well-formed activity is accepted
	activity := mapof.Any{
		vocab.PropertyID:     "https://example.com/@alice#updates/cafebabe",
		vocab.PropertyType:   vocab.ActivityTypeUpdate,
		vocab.PropertyActor:  "https://example.com/@alice",
		vocab.PropertyCC:     []any{"https://example.com/@alice/pub/followers"},
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: "https://example.com/@alice"},
	}

	require.NoError(t, service.Send(session, activity))
}

// newOutbox2PublishTest returns an Outbox2 on example.com and a mock session that spools every task it publishes
func newOutbox2PublishTest(t *testing.T) (*Outbox2, data.Session, *postcommit.Tasks) {

	t.Helper()

	locator := Locator{host: "https://example.com"}
	outbox2 := Outbox2{locator: &locator, host: "https://example.com"}
	spool := postcommit.NewTasks()

	session, err := mockdb.New().Session(postcommit.WithContext(context.Background(), spool))
	require.NoError(t, err)

	return &outbox2, session, spool
}

// countOutbox2 returns the number of rows in the Outbox2 collection
func countOutbox2(t *testing.T, session data.Session) int64 {

	t.Helper()

	count, err := session.Collection("Outbox2").Count(exp.All())
	require.NoError(t, err)
	return count
}

// TestOutbox2_Save_StoresOnly verifies that Save writes the row and queues no delivery
func TestOutbox2_Save_StoresOnly(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	item := model.NewOutboxItem()
	item.ActorType = model.ActorTypeUser
	item.ActorID = primitive.NewObjectID()
	item.Activity = mapof.Any{
		vocab.PropertyType: vocab.ActivityTypeCreate,
		vocab.PropertyTo:   []any{"https://remote.example/@bob"},
	}

	require.NoError(t, outbox2.Save(session, &item, "test"))
	require.Equal(t, int64(1), countOutbox2(t, session))
	require.Empty(t, spool.Drain(), "Save must never deliver; an import stores history through it")
}

// TestOutbox2_Publish_MintsAndQueues verifies that Publish mints the activity's URL, records what
// the activity is about, and queues one delivery task that names the stored row
func TestOutbox2_Publish_MintsAndQueues(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)
	userID := primitive.NewObjectID()
	permissions := model.Permissions{primitive.NewObjectID()}

	activity := streams.NewDocument(mapof.Any{
		vocab.PropertyType:   vocab.ActivityTypeCreate,
		vocab.PropertyActor:  "https://example.com/@" + userID.Hex(),
		vocab.PropertyTo:     []any{"https://remote.example/@bob"},
		vocab.PropertyBCC:    []any{"https://remote.example/@carol"},
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: "https://example.com/notes/1", vocab.PropertyType: vocab.ObjectTypeNote},
	})

	activityURL, err := outbox2.Publish(session, model.ActorTypeUser, userID, activity, permissions)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(activityURL, "https://example.com/@"+userID.Hex()+"/pub/outbox/"), activityURL)
	require.Equal(t, activityURL, activity.ID(), "the caller's activity carries the minted id")

	// The row describes the activity
	item := model.NewOutboxItem()
	require.NoError(t, session.Collection("Outbox2").Load(exp.Equal("url", activityURL), &item))
	require.Equal(t, model.ActorTypeUser, item.ActorType)
	require.Equal(t, userID, item.ActorID)
	require.Equal(t, vocab.ActivityTypeCreate, item.ActivityType)
	require.Equal(t, "https://example.com/notes/1", item.ObjectID)
	require.Equal(t, permissions, item.Permissions)
	require.ElementsMatch(t, []string{"https://remote.example/@bob", "https://remote.example/@carol"}, []string(item.Recipients))
	require.Equal(t, activityURL, item.Activity.GetString(vocab.PropertyID))

	// One task fans it out, after the commit, and carries the activity with its blind recipients
	tasks := spool.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, delivery.TaskFanOut, tasks[0].Name)
	require.Equal(t, "example.com", tasks[0].Arguments.GetString("hostname"))
	require.Equal(t, "https://example.com/@"+userID.Hex(), tasks[0].Arguments.GetString("actor"))

	queued := tasks[0].Arguments.GetMap("activity")
	require.Equal(t, activityURL, queued.GetString(vocab.PropertyID))
	require.Equal(t, []string{"https://remote.example/@carol"}, queued.GetSliceOfString(vocab.PropertyBCC))
}

// TestOutbox2_Publish_EmbeddedActor verifies that the delivery task names the sending actor
// when the activity embeds that actor as an object, because the worker builds its pipeline,
// its followers and its rules from that name
func TestOutbox2_Publish_EmbeddedActor(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)
	userID := primitive.NewObjectID()
	actorURL := "https://example.com/@" + userID.Hex()

	activity := streams.NewDocument(mapof.Any{
		vocab.PropertyType:   vocab.ActivityTypeCreate,
		vocab.PropertyActor:  mapof.Any{vocab.PropertyID: actorURL, vocab.PropertyType: vocab.ActorTypePerson},
		vocab.PropertyTo:     []any{"https://remote.example/@bob"},
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: "https://example.com/notes/1", vocab.PropertyType: vocab.ObjectTypeNote},
	})

	_, err := outbox2.Publish(session, model.ActorTypeUser, userID, activity, model.NewPermissions())
	require.NoError(t, err)

	tasks := spool.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, actorURL, tasks[0].Arguments.GetString("actor"))
}

// TestOutbox2_Publish_RequiresActor verifies that an activity which names no actor is refused
// before anything is stored or queued, because nothing could sign or route its delivery
func TestOutbox2_Publish_RequiresActor(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)
	userID := primitive.NewObjectID()

	for name, actor := range map[string]any{
		"missing":           nil,
		"empty string":      "",
		"object without id": mapof.Any{vocab.PropertyType: vocab.ActorTypePerson, vocab.PropertyName: "Sally"},
		"empty list":        []any{},
	} {
		t.Run(name, func(t *testing.T) {

			activity := mapof.Any{
				vocab.PropertyType:   vocab.ActivityTypeCreate,
				vocab.PropertyTo:     []any{"https://remote.example/@bob"},
				vocab.PropertyBCC:    []any{"https://remote.example/@secret"},
				vocab.PropertyObject: mapof.Any{vocab.PropertyID: "https://example.com/notes/1", vocab.PropertyType: vocab.ObjectTypeNote},
			}

			if actor != nil {
				activity[vocab.PropertyActor] = actor
			}

			activityURL, err := outbox2.Publish(session, model.ActorTypeUser, userID, streams.NewDocument(activity), model.NewPermissions())
			require.Error(t, err)
			require.True(t, derp.IsInternalServerError(err))
			require.Empty(t, activityURL)
			require.NotContains(t, err.Error(), "@secret", "the error names the actor, and does not carry the activity")

			// Nothing was stored, and nothing waits on the commit
			count, err := outbox2.Count(session, exp.All())
			require.NoError(t, err)
			require.Zero(t, count)
			require.Empty(t, spool.Drain())
		})
	}
}

// TestOutbox2_QueueDelivery_RequiresActor verifies that the last step before the queue makes
// its own check, so that no caller can queue a fan-out that names no actor
func TestOutbox2_QueueDelivery_RequiresActor(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	err := outbox2.queueDelivery(session, streams.NewDocument(mapof.Any{
		vocab.PropertyID:   "https://example.com/activities/1",
		vocab.PropertyType: vocab.ActivityTypeCreate,
		vocab.PropertyBCC:  []any{"https://remote.example/@secret"},
	}))

	require.Error(t, err)
	require.True(t, derp.IsInternalServerError(err))
	require.NotContains(t, err.Error(), "@secret")
	require.Empty(t, spool.Drain())
}

// TestOutbox2_PutUserActivity_IgnoresClientIDs verifies that a client cannot choose the URL of
// its activity, in the `id` spelling or in JSON-LD's `@id`
func TestOutbox2_PutUserActivity_IgnoresClientIDs(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)
	userID := primitive.NewObjectID()
	actorURL := "https://example.com/@" + userID.Hex()

	for name, ids := range map[string]mapof.Any{
		"id":       {vocab.PropertyID: "https://victim.example/activities/1"},
		"@id":      {vocab.PropertyID_Alternate: "https://victim.example/activities/2"},
		"both":     {vocab.PropertyID: "urn:uuid:1", vocab.PropertyID_Alternate: "https://victim.example/activities/3"},
		"neither":  {},
		"empty id": {vocab.PropertyID: "", vocab.PropertyID_Alternate: "https://example.com/@someone-else/pub/outbox/4"},
	} {
		t.Run(name, func(t *testing.T) {

			activity := mapof.Any{
				vocab.PropertyType:   vocab.ActivityTypeCreate,
				vocab.PropertyActor:  actorURL,
				vocab.PropertyTo:     []any{"https://remote.example/@bob"},
				vocab.PropertyObject: mapof.Any{vocab.PropertyType: vocab.ObjectTypeNote, vocab.PropertyContent: "hello"},
			}

			for key, value := range ids {
				activity[key] = value
			}

			activityURL, err := outbox2.PutUserActivity(session, userID, streams.NewDocument(activity))
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(activityURL, actorURL+"/pub/outbox/"), activityURL)

			tasks := spool.Drain()
			require.Len(t, tasks, 1)

			queued := tasks[0].Arguments.GetMap("activity")
			require.Equal(t, activityURL, queued.GetString(vocab.PropertyID))
			require.NotContains(t, queued, vocab.PropertyID_Alternate)
		})
	}
}

// TestOutbox2_Publish_KeepsCarriedInID verifies that an activity with its own canonical id keeps
// it, and that publishing the same id again stores and sends nothing more
func TestOutbox2_Publish_KeepsCarriedInID(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)
	userID := primitive.NewObjectID()
	const likeURL = "https://example.com/@sender/pub/liked/1"

	newLike := func() streams.Document {
		return streams.NewDocument(mapof.Any{
			vocab.PropertyID:     likeURL,
			vocab.PropertyType:   vocab.ActivityTypeLike,
			vocab.PropertyActor:  "https://example.com/@" + userID.Hex(),
			vocab.PropertyTo:     []any{"https://remote.example/@author"},
			vocab.PropertyObject: "https://remote.example/notes/1",
		})
	}

	activityURL, err := outbox2.Publish(session, model.ActorTypeUser, userID, newLike(), model.NewAnonymousPermissions())
	require.NoError(t, err)
	require.Equal(t, likeURL, activityURL)
	require.Len(t, spool.Drain(), 1)

	activityURL, err = outbox2.Publish(session, model.ActorTypeUser, userID, newLike(), model.NewAnonymousPermissions())
	require.NoError(t, err)
	require.Equal(t, likeURL, activityURL)
	require.Equal(t, int64(1), countOutbox2(t, session), "a replayed id must not store a second row")
	require.Empty(t, spool.Drain(), "a replayed id must not be delivered again")
}

// TestOutbox2_Publish_UnknownActorType verifies that an activity no peer could dereference is refused
func TestOutbox2_Publish_UnknownActorType(t *testing.T) {

	outbox2, session, spool := newOutbox2PublishTest(t)

	activity := streams.NewDocument(mapof.Any{vocab.PropertyType: vocab.ActivityTypeCreate})

	_, err := outbox2.Publish(session, "Unknown", primitive.NewObjectID(), activity, model.NewPermissions())
	require.Error(t, err)
	require.Zero(t, countOutbox2(t, session))
	require.Empty(t, spool.Drain())
}

// TestOutbox2_PutUserActivity_UndoActor runs the BUG-62 rule through PutUserActivity, so the test
// fails if the check is ever disconnected from the C2S path
func TestOutbox2_PutUserActivity_UndoActor(t *testing.T) {

	userID := primitive.NewObjectID()
	me := "https://example.com/@" + userID.Hex()
	const bob = "https://remote.example/@bob"

	// Every document loads through this client, so a test can prove that nothing was fetched
	client := &countingClient{}

	undo := func(object any) streams.Document {
		return streams.NewDocument(mapof.Any{
			vocab.PropertyType:   vocab.ActivityTypeUndo,
			vocab.PropertyActor:  me,
			vocab.PropertyTo:     []any{bob},
			vocab.PropertyObject: object,
		}, streams.WithClient(client))
	}

	like := func(actor string) map[string]any {
		return map[string]any{
			vocab.PropertyID:     me + "/pub/liked/1",
			vocab.PropertyType:   vocab.ActivityTypeLike,
			vocab.PropertyActor:  actor,
			vocab.PropertyObject: "https://remote.example/notes/1",
		}
	}

	t.Run("another actor's activity is refused", func(t *testing.T) {
		outbox2, session, spool := newOutbox2PublishTest(t)

		_, err := outbox2.PutUserActivity(session, userID, undo(like("https://other.example/@victim")))
		require.Error(t, err)
		require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))
		require.Zero(t, countOutbox2(t, session))
		require.Empty(t, spool.Drain())
	})

	t.Run("the actor's own activity is published", func(t *testing.T) {
		outbox2, session, spool := newOutbox2PublishTest(t)

		_, err := outbox2.PutUserActivity(session, userID, undo(like(me)))
		require.NoError(t, err)
		require.Equal(t, int64(1), countOutbox2(t, session))
		require.Len(t, spool.Drain(), 1)
	})

	t.Run("a bare URL is loaded, and published when the activity is the actor's own", func(t *testing.T) {
		outbox2, session, spool := newOutbox2PublishTest(t)

		client.loads = 0
		client.documents = map[string]map[string]any{me + "/pub/liked/1": like(me)}

		_, err := outbox2.PutUserActivity(session, userID, undo(me+"/pub/liked/1"))
		require.NoError(t, err)
		require.Equal(t, 1, client.loads)
		require.Equal(t, int64(1), countOutbox2(t, session))
		require.Len(t, spool.Drain(), 1)
	})

	t.Run("a bare URL that names another actor's activity is refused", func(t *testing.T) {
		outbox2, session, spool := newOutbox2PublishTest(t)

		client.documents = map[string]map[string]any{"https://other.example.org/likes/1": like("https://other.example/@victim")}

		_, err := outbox2.PutUserActivity(session, userID, undo("https://other.example.org/likes/1"))
		require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))
		require.Zero(t, countOutbox2(t, session))
		require.Empty(t, spool.Drain())
	})

	t.Run("a bare URL that cannot be loaded is refused", func(t *testing.T) {
		outbox2, session, spool := newOutbox2PublishTest(t)

		client.documents = nil

		_, err := outbox2.PutUserActivity(session, userID, undo(me+"/pub/liked/1"))
		require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))
		require.Zero(t, countOutbox2(t, session))
		require.Empty(t, spool.Drain())
	})

	t.Run("an activity sent as someone else is refused", func(t *testing.T) {
		outbox2, session, _ := newOutbox2PublishTest(t)

		activity := undo(like(me))
		activity.SetProperty(vocab.PropertyActor, "https://example.com/@"+primitive.NewObjectID().Hex())

		_, err := outbox2.PutUserActivity(session, userID, activity)
		require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))
		require.Zero(t, countOutbox2(t, session))
	})

	t.Run("an activity that is not an Undo is unaffected", func(t *testing.T) {
		outbox2, session, _ := newOutbox2PublishTest(t)

		announce := undo(like("https://other.example/@victim"))
		announce.SetProperty(vocab.PropertyType, vocab.ActivityTypeAnnounce)

		_, err := outbox2.PutUserActivity(session, userID, announce)
		require.NoError(t, err)
		require.Equal(t, int64(1), countOutbox2(t, session))
	})

}

// recordingEmailer is a followerEmailer that records each Follower it sends to, and what it sent
type recordingEmailer struct {
	sent       []string
	activities []mapof.Any
}

// IsConfigured reports that email can be sent
func (emailer *recordingEmailer) IsConfigured() bool {
	return true
}

// SendFollowerActivity records the Follower's address and the activity, and sends nothing
func (emailer *recordingEmailer) SendFollowerActivity(follower *model.Follower, activity mapof.Any) error {
	emailer.sent = append(emailer.sent, follower.Actor.EmailAddress)
	emailer.activities = append(emailer.activities, activity)
	return nil
}

// outbox2DeliverTest holds an Outbox2 wired to in-memory followers, rules, and identities
type outbox2DeliverTest struct {
	outbox2  Outbox2
	session  deliverSession
	spool    *postcommit.Tasks
	emailer  *recordingEmailer
	userID   primitive.ObjectID
	actorURL string
}

// newOutbox2DeliverTest returns a delivery fixture for a User on localhost with the provided followers.
// The activity's actor names a public host, so the same-network rule skips it, and no inbox is looked up.
func newOutbox2DeliverTest(t *testing.T, followers ...model.Follower) outbox2DeliverTest {

	t.Helper()

	userID := primitive.NewObjectID()

	for index := range followers {
		followers[index].ParentID = userID
		followers[index].ParentType = model.FollowerTypeUser
	}

	ruleService, _ := newRuleService(&ruleStore{})
	spool := postcommit.NewTasks()
	emailer := &recordingEmailer{}

	return outbox2DeliverTest{
		outbox2: Outbox2{
			followerService: &Follower{},
			identityService: &Identity{},
			ruleService:     ruleService,
			domainEmail:     emailer,
			host:            "http://localhost",
		},
		session: deliverSession{
			context: postcommit.WithContext(context.Background(), spool),
			collections: map[string]data.Collection{
				"Follower": &followerCollection{records: followers},
				"Identity": identityByActor{identities: map[string]model.Identity{}},
				"Rule":     &ruleStore{},
			},
		},
		spool:    spool,
		emailer:  emailer,
		userID:   userID,
		actorURL: "https://sender.example/@" + userID.Hex(),
	}
}

// item returns a stored activity from the fixture's User, with the provided addressing and permissions
func (test outbox2DeliverTest) item(permissions model.Permissions, to ...any) model.OutboxItem {

	item := model.NewOutboxItem()
	item.ActorType = model.ActorTypeUser
	item.ActorID = test.userID
	item.Permissions = permissions
	item.Activity = mapof.Any{
		vocab.PropertyID:     test.actorURL + "/pub/outbox/" + item.ActivityID.Hex(),
		vocab.PropertyType:   vocab.ActivityTypeCreate,
		vocab.PropertyActor:  test.actorURL,
		vocab.PropertyTo:     to,
		vocab.PropertyBCC:    []any{"https://blind.example/@bcc"},
		vocab.PropertyObject: mapof.Any{vocab.PropertyID: "http://localhost/notes/1"},
	}
	item.CalcRecipients()

	return item
}

// deliver runs the follower loop over every Follower of the fixture's User
func (test outbox2DeliverTest) deliver(t *testing.T, item *model.OutboxItem) {

	t.Helper()

	loop := followerDelivery{
		identityService: test.outbox2.identityService,
		ruleService:     test.outbox2.ruleService,
		domainEmail:     test.emailer,
		host:            test.outbox2.host,
	}

	recipients := test.outbox2.followerService.RangeFollowers(test.session, model.ActorTypeUser, test.userID)
	require.NoError(t, loop.deliver(test.session, item.ActorType, item.ActorID, item.Activity, item.Permissions, recipients))
}

// followers returns the URI of the fixture User's followers collection
func (test outbox2DeliverTest) followers() string {
	return "http://localhost/@" + test.userID.Hex() + "/pub/followers"
}

// inboxes returns the inbox of every single-recipient delivery task, and fails on any other task
func (test outbox2DeliverTest) inboxes(t *testing.T) []string {

	t.Helper()

	result := make([]string, 0)

	for _, task := range test.spool.Drain() {
		require.Equal(t, sender.OutboxSendToSingleRecipient, task.Name)
		result = append(result, task.Arguments.GetString("inbox"))
	}

	return result
}

// newTestFollower returns an active Follower on localhost who receives by the provided method
func newTestFollower(method string, name string) model.Follower {

	follower := model.NewFollower()
	follower.FollowerID = primitive.NewObjectID()
	follower.Method = method
	follower.StateID = model.FollowerStateActive
	follower.Actor.ProfileURL = "http://localhost/@" + name
	follower.Actor.InboxURL = "http://localhost/@" + name + "/pub/inbox"
	follower.Actor.EmailAddress = name + "@localhost"

	return follower
}

// TestFollowerDelivery_ToFollowers verifies that the loop the legacy outbox uses reaches each
// ActivityPub follower once, and emails only confirmed email followers
func TestFollowerDelivery_ToFollowers(t *testing.T) {

	pending := newTestFollower(model.FollowerMethodEmail, "pending")
	pending.StateID = model.FollowerStatePending

	test := newOutbox2DeliverTest(t,
		newTestFollower(model.FollowerMethodActivityPub, "alice"),
		newTestFollower(model.FollowerMethodEmail, "carol"),
		pending,
	)

	item := test.item(model.NewAnonymousPermissions(), vocab.NamespaceActivityStreamsPublic, test.followers(), "https://remote.example/@dave")
	test.deliver(t, &item)

	// The follower receives one delivery; the actor and addressees are on another network
	require.Equal(t, []string{"http://localhost/@alice/pub/inbox"}, test.inboxes(t))

	// RULE: Email only confirmed Followers (BUG-207 item 2)
	require.Equal(t, []string{"carol@localhost"}, test.emailer.sent)

	// RULE: An email carries the stripped activity, like every other delivery
	require.NotContains(t, test.emailer.activities[0], vocab.PropertyBCC)
	require.Contains(t, item.Activity, vocab.PropertyBCC, "the stored activity keeps its addressing")
}

// TestFollowerDelivery_StripsNestedBlindRecipients verifies that the loop the legacy outbox uses
// removes `bto` and `bcc` from the embedded object too, and leaves the stored activity whole
func TestFollowerDelivery_StripsNestedBlindRecipients(t *testing.T) {

	test := newOutbox2DeliverTest(t,
		newTestFollower(model.FollowerMethodActivityPub, "alice"),
		newTestFollower(model.FollowerMethodEmail, "carol"),
	)

	item := test.item(model.NewAnonymousPermissions(), vocab.NamespaceActivityStreamsPublic, test.followers())
	item.Activity[vocab.PropertyObject] = mapof.Any{
		vocab.PropertyID:  "http://localhost/notes/1",
		vocab.PropertyBTo: []any{"https://blind.example/@nested-bto"},
		vocab.PropertyTag: []any{mapof.Any{vocab.PropertyBCC: "https://blind.example/@nested-bcc"}},
	}

	test.deliver(t, &item)

	// RULE: The body every ActivityPub recipient receives names no blind recipient at any level
	tasks := test.spool.Drain()
	require.Len(t, tasks, 1)

	body := tasks[0].Arguments.GetString("body")
	require.Contains(t, body, "http://localhost/notes/1")
	require.NotContains(t, body, "blind.example")

	// RULE: An email carries the same stripped activity
	require.Len(t, test.emailer.activities, 1)

	emailed, err := json.Marshal(test.emailer.activities[0])
	require.NoError(t, err)
	require.NotContains(t, string(emailed), "blind.example")

	// The stored activity keeps its addressing, because delivery reads it
	stored, err := json.Marshal(item.Activity)
	require.NoError(t, err)
	require.Contains(t, string(stored), "@nested-bto")
	require.Contains(t, string(stored), "@nested-bcc")
	require.Contains(t, item.Activity, vocab.PropertyBCC)
}

// TestFollowerDelivery_CircleOnly verifies that the loop the legacy outbox uses sends nothing to a
// follower outside the activity's audience
func TestFollowerDelivery_CircleOnly(t *testing.T) {

	test := newOutbox2DeliverTest(t,
		newTestFollower(model.FollowerMethodActivityPub, "member"),
		newTestFollower(model.FollowerMethodActivityPub, "outsider"),
		newTestFollower(model.FollowerMethodEmail, "subscriber"),
	)

	// Only "member" holds the circle's privilege
	circleID := primitive.NewObjectID()
	identity := model.NewIdentity()
	identity.ActivityPubActor = "http://localhost/@member"
	identity.PrivilegeIDs = id.Slice{circleID}
	test.session.collections["Identity"] = identityByActor{identities: map[string]model.Identity{identity.ActivityPubActor: identity}}

	item := test.item(model.Permissions{circleID}, test.followers())
	test.deliver(t, &item)

	require.Equal(t, []string{"http://localhost/@member/pub/inbox"}, test.inboxes(t))
	require.Empty(t, test.emailer.sent, "a circle-only post sends no email to a follower outside the circle")
}

// identityByActor is a data.Collection that finds an Identity by its ActivityPub actor. data-mock cannot
// stand in, because it never matches the `deleteDate: 0` clause that Identity.Load adds. Only Load is used.
type identityByActor struct {
	data.Collection
	identities map[string]model.Identity
}

// Load copies the Identity whose actor the criteria name into the target, or answers not found
func (collection identityByActor) Load(criteria exp.Expression, target data.Object, _ ...option.Option) error {

	for actor, identity := range collection.identities {

		// Match the actor, and accept the "not deleted" clause that every Identity load adds
		if criteria.Match(func(predicate exp.Predicate) bool {
			switch predicate.Field {
			case "activityPubActor":
				return predicate.Value == actor
			case "deleteDate":
				return true
			}
			return false
		}) {
			*target.(*model.Identity) = identity
			return nil
		}
	}

	return derp.NotFound("identityByActor.Load", "Identity not found")
}

// TestOutbox2_requireUndoActorMatches verifies that an Undo may reverse only its own actor's
// activities: every object it names must state that actor, and one named by URL is loaded to find out
func TestOutbox2_requireUndoActorMatches(t *testing.T) {

	// BUG-62: the outbox signed and delivered an Undo of another actor's Like (ActivityPub §6.10)
	const me = "https://example.com/@sender"
	const someoneElse = "https://other.example.org/@victim"
	const myLikeURL = "https://example.com/@sender/pub/liked/1"
	const theirLikeURL = "https://other.example.org/@victim/pub/liked/1"
	const noteURL = "https://example.com/notes/1"
	const missingURL = "https://example.com/@sender/pub/liked/404"

	like := func(actor any) map[string]any {
		return map[string]any{
			vocab.PropertyID:     myLikeURL,
			vocab.PropertyType:   vocab.ActivityTypeLike,
			vocab.PropertyActor:  actor,
			vocab.PropertyObject: "https://other.example/notes/1",
		}
	}

	// Every document loads through this client, so each case can say how many it fetched
	client := &countingClient{documents: map[string]map[string]any{
		myLikeURL:    like(me),
		theirLikeURL: like(someoneElse),
		noteURL:      {vocab.PropertyID: noteURL, vocab.PropertyType: vocab.ObjectTypeNote},
	}}

	undoAs := func(actor any, activityType any, object any) streams.Document {

		value := mapof.Any{vocab.PropertyType: activityType}

		if actor != nil {
			value[vocab.PropertyActor] = actor
		}

		if object != nil {
			value[vocab.PropertyObject] = object
		}

		return streams.NewDocument(value, streams.WithClient(client))
	}

	undo := func(object any) streams.Document {
		return undoAs(me, vocab.ActivityTypeUndo, object)
	}

	person := map[string]any{vocab.PropertyType: vocab.ActorTypePerson}

	for name, test := range map[string]struct {
		activity  streams.Document
		forbidden bool
		loads     int
	}{
		// Embedded activities are read as they are, and never loaded
		"own activity":                         {undo(like(me)), false, 0},
		"own activity, actor embedded":         {undo(like(map[string]any{vocab.PropertyID: me})), false, 0},
		"another actor's activity":             {undo(like(someoneElse)), true, 0},
		"another actor, embedded":              {undo(like(map[string]any{vocab.PropertyID: someoneElse})), true, 0},
		"several objects, all the actor's own": {undo([]any{like(me), like(me)}), false, 0},
		"one of several objects is not":        {undo([]any{like(me), like(someoneElse)}), true, 0},
		"first of several objects is not":      {undo([]any{like(someoneElse), like(me)}), true, 0},

		// RULE: The first actor of a list decides, as it does for the Undo itself
		"first of several actors is the actor": {undo(like([]any{me, someoneElse})), false, 0},
		"first of several actors is not":       {undo(like([]any{someoneElse, me})), true, 0},
		"first actor has no id":                {undo(like([]any{person, me})), true, 0},

		// RULE: An object that states no actor is refused, and not left to the receiver
		"embedded object with no actor": {undo(map[string]any{vocab.PropertyType: vocab.ActivityTypeLike}), true, 0},
		"embedded actor with no id":     {undo(like(person)), true, 0},
		"embedded actor is empty":       {undo(like("")), true, 0},
		"object named only by an id":    {undo(map[string]any{vocab.PropertyID: myLikeURL}), true, 0},
		"object is not a document":      {undo(42), true, 0},

		// RULE: An object named by its URL is loaded, and the activity found there is checked
		"bare URL of the actor's own activity":  {undo(myLikeURL), false, 1},
		"bare URL of another actor's activity":  {undo(theirLikeURL), true, 1},
		"bare URL that cannot be loaded":        {undo(missingURL), true, 1},
		"bare string that is not a URL":         {undo("not a url"), true, 0},
		"bare URL of an object with no actor":   {undo(noteURL), true, 1},
		"bare URL beside an embedded activity":  {undo([]any{myLikeURL, like(me)}), false, 1},
		"bare URL of another, beside the own":   {undo([]any{like(me), theirLikeURL}), true, 1},
		"the check stops at the first mismatch": {undo([]any{like(someoneElse), myLikeURL}), true, 0},

		// RULE: Any spelling of the Undo type is checked
		"Undo listed first":                {undoAs(me, []any{vocab.ActivityTypeUndo, "Other"}, like(someoneElse)), true, 0},
		"Undo listed second":               {undoAs(me, []any{"Other", vocab.ActivityTypeUndo}, like(someoneElse)), true, 0},
		"Undo listed second, own activity": {undoAs(me, []any{"Other", vocab.ActivityTypeUndo}, like(me)), false, 0},

		// RULE: The Undo itself must name its actor
		"Undo with no actor":         {undoAs(nil, vocab.ActivityTypeUndo, like(me)), true, 0},
		"Undo with an empty actor":   {undoAs("", vocab.ActivityTypeUndo, like("")), true, 0},
		"Undo whose actor has no id": {undoAs(person, vocab.ActivityTypeUndo, like(person)), true, 0},

		// Nothing else is this function's business
		"no object":                     {undo(nil), false, 0},
		"not an Undo":                   {undoAs(me, vocab.ActivityTypeAnnounce, like(someoneElse)), false, 0},
		"not an Undo, object by URL":    {undoAs(me, vocab.ActivityTypeAnnounce, theirLikeURL), false, 0},
		"type is spelled in lower case": {undoAs(me, "undo", like(someoneElse)), false, 0},
		"no type":                       {undoAs(me, nil, like(someoneElse)), false, 0},
	} {
		t.Run(name, func(t *testing.T) {

			client.loads = 0

			err := requireUndoActorMatches(test.activity)

			if test.forbidden {
				require.Error(t, err)
				require.Equal(t, http.StatusForbidden, derp.ErrorCode(err))
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.loads, client.loads, "documents loaded")
		})
	}
}

// countingClient is a streams.Client that counts each Load. It serves the documents it holds,
// and refuses every other URL.
type countingClient struct {
	loads     int
	documents map[string]map[string]any
}

// SetRootClient does nothing, because this client wraps no other
func (client *countingClient) SetRootClient(streams.Client) {}

// Load records the attempt, then returns the document held for the URL or refuses it
func (client *countingClient) Load(uri string, _ ...any) (streams.Document, error) {

	client.loads++

	if document, found := client.documents[uri]; found {
		return streams.NewDocument(document, streams.WithClient(client)), nil
	}

	return streams.NilDocument(), derp.Internal("countingClient.Load", "Loading is not allowed in this test", uri)
}

// Save does nothing, because nothing in these tests writes to the cache
func (client *countingClient) Save(streams.Document) error {
	return nil
}

// Delete does nothing, because nothing in these tests writes to the cache
func (client *countingClient) Delete(string) error {
	return nil
}
