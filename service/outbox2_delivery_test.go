package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/floodgate"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These tests run the delivery adapters against a real MongoDB, because they are queries over
// Followers, Circles and Privileges. They skip when no database is reachable.

// deliveryTest holds the collections adapter for one User on example.com
type deliveryTest struct {
	session     data.Session
	collections deliveryCollections
	locator     *Locator
	userID      primitive.ObjectID
}

// newDeliveryTest returns the collections adapter for a new User, over an empty database
func newDeliveryTest(t *testing.T) deliveryTest {

	t.Helper()

	query := newOutbox2QueryTest(t)
	locator := &Locator{host: "https://example.com"}
	userID := primitive.NewObjectID()

	return deliveryTest{
		session: query.session,
		locator: locator,
		userID:  userID,
		collections: deliveryCollections{
			session:          query.session,
			host:             "https://example.com",
			locator:          locator,
			followerService:  &Follower{},
			circleService:    &Circle{},
			privilegeService: &Privilege{},
			identityService:  &Identity{},
			actorType:        model.ActorTypeUser,
			actorID:          userID,
		},
	}
}

// followers returns the URL of an actor's followers collection
func (test deliveryTest) followers(actorType string, actorID primitive.ObjectID) string {
	return test.locator.FollowersURL(actorType, actorID)
}

// addFollower saves one Follower of the given actor
func (test deliveryTest) addFollower(t *testing.T, parentType string, parentID primitive.ObjectID, method string, name string, stateID string) {

	t.Helper()

	follower := model.NewFollower()
	follower.ParentType = parentType
	follower.ParentID = parentID
	follower.Method = method
	follower.StateID = stateID
	follower.Actor.ProfileURL = "https://remote.example/@" + name
	follower.Actor.EmailAddress = name + "@remote.example"

	require.NoError(t, test.session.Collection("Follower").Save(&follower, "test"))
}

// addCircle saves a circle owned by the given User, and returns its ID
func (test deliveryTest) addCircle(t *testing.T, userID primitive.ObjectID) primitive.ObjectID {

	t.Helper()

	circle := model.NewCircle()
	circle.UserID = userID
	circle.Name = "Test Circle"

	require.NoError(t, test.session.Collection("Circle").Save(&circle, "test"))
	return circle.CircleID
}

// addMember saves one member of a circle, known by the given identifier, and returns the ID of their Identity
func (test deliveryTest) addMember(t *testing.T, circleID primitive.ObjectID, identifierType string, identifierValue string, identity *model.Identity) {

	t.Helper()

	privilege := model.NewPrivilege()
	privilege.UserID = test.userID
	privilege.CircleID = circleID
	privilege.IdentifierType = identifierType
	privilege.IdentifierValue = identifierValue

	if identity != nil {
		require.NoError(t, test.session.Collection("Identity").Save(identity, "test"))
		privilege.IdentityID = identity.IdentityID
	}

	require.NoError(t, test.session.Collection("Privilege").Save(&privilege, "test"))
}

// members returns the sorted members of a collection
func (test deliveryTest) members(t *testing.T, collectionURL string) []string {

	t.Helper()

	members, err := test.collections.Members(collectionURL)
	require.NoError(t, err)

	result := slices.Collect(members)
	slices.Sort(result)
	return slices.DeleteFunc(result, func(member string) bool { return member == "" })
}

// TestDeliveryCollections_IsCollection verifies that every followers collection and circle on
// this server is recognized, and nothing else
func TestDeliveryCollections_IsCollection(t *testing.T) {

	test := newDeliveryTest(t)
	other := primitive.NewObjectID()

	require.True(t, test.collections.IsCollection(test.followers(model.ActorTypeUser, test.userID)))
	require.True(t, test.collections.IsCollection(test.followers(model.ActorTypeUser, other)), "another actor's collection is still a collection")
	require.True(t, test.collections.IsCollection(test.followers(model.ActorTypeStream, other)))
	require.True(t, test.collections.IsCollection(test.locator.CircleURL(test.userID, other)))

	require.False(t, test.collections.IsCollection(test.locator.UserURL(test.userID)))
	require.False(t, test.collections.IsCollection("https://remote.example/@alice/followers"))
	require.False(t, test.collections.IsCollection("https://remote.example/@"+test.userID.Hex()+"/pub/followers"))
	require.False(t, test.collections.IsCollection("mailto:someone@example.com"))
}

// TestDeliveryCollections_Followers verifies who an actor's followers collection expands to
func TestDeliveryCollections_Followers(t *testing.T) {

	test := newDeliveryTest(t)
	other := primitive.NewObjectID()

	test.addFollower(t, model.FollowerTypeUser, test.userID, model.FollowerMethodActivityPub, "alice", model.FollowerStateActive)
	test.addFollower(t, model.FollowerTypeUser, test.userID, model.FollowerMethodActivityPub, "blocked", model.FollowerStateBlocked)
	test.addFollower(t, model.FollowerTypeUser, test.userID, model.FollowerMethodEmail, "carol", model.FollowerStateActive)
	test.addFollower(t, model.FollowerTypeUser, test.userID, model.FollowerMethodEmail, "pending", model.FollowerStatePending)
	test.addFollower(t, model.FollowerTypeUser, other, model.FollowerMethodActivityPub, "stranger", model.FollowerStateActive)
	test.addFollower(t, model.FollowerTypeStream, test.userID, model.FollowerMethodActivityPub, "streamfan", model.FollowerStateActive)

	// The sender's own followers: ActivityPub by actor, confirmed email by address
	require.Equal(t,
		[]string{"https://remote.example/@alice", "mailto:carol@remote.example"},
		test.members(t, test.followers(model.ActorTypeUser, test.userID)),
		"a blocked follower and an unconfirmed email follower receive nothing",
	)

	// RULE: Naming another actor's followers adds no one
	require.Empty(t, test.members(t, test.followers(model.ActorTypeUser, other)))
	require.Empty(t, test.members(t, test.followers(model.ActorTypeStream, test.userID)), "a Stream with the same ID is a different actor")
}

// TestDeliveryCollections_Circle verifies that a circle expands to every member, follower or
// not: by ActivityPub when the member has an actor, and by email otherwise
func TestDeliveryCollections_Circle(t *testing.T) {

	test := newDeliveryTest(t)
	circleID := test.addCircle(t, test.userID)

	// A member who joined as an actor
	test.addMember(t, circleID, model.IdentifierTypeActivityPub, "https://remote.example/@member", nil)

	// A member who joined by email, and has no actor
	emailOnly := model.NewIdentity()
	emailOnly.EmailAddress = "email-only@remote.example"
	test.addMember(t, circleID, model.IdentifierTypeEmail, "email-only@remote.example", &emailOnly)

	// A member who joined by email, and also has an actor
	both := model.NewIdentity()
	both.EmailAddress = "both@remote.example"
	both.ActivityPubActor = "https://remote.example/@both"
	test.addMember(t, circleID, model.IdentifierTypeEmail, "both@remote.example", &both)

	// A member of some other circle
	test.addMember(t, primitive.NewObjectID(), model.IdentifierTypeActivityPub, "https://remote.example/@elsewhere", nil)

	require.Equal(t,
		[]string{"https://remote.example/@both", "https://remote.example/@member", "mailto:email-only@remote.example"},
		test.members(t, test.locator.CircleURL(test.userID, circleID)),
	)
}

// TestDeliveryCollections_Circle_NotTheSenders verifies that a circle reaches no one unless the
// sending User owns it and it still exists
func TestDeliveryCollections_Circle_NotTheSenders(t *testing.T) {

	test := newDeliveryTest(t)
	other := primitive.NewObjectID()

	// Another User's circle, with a member
	foreign := test.addCircle(t, other)
	test.addMember(t, foreign, model.IdentifierTypeActivityPub, "https://remote.example/@theirs", nil)

	require.Empty(t, test.members(t, test.locator.CircleURL(other, foreign)), "another User's circle")
	require.Empty(t, test.members(t, test.locator.CircleURL(test.userID, foreign)), "a circle that this User does not own, named under their own URL")
	require.Empty(t, test.members(t, test.locator.CircleURL(test.userID, primitive.NewObjectID())), "a circle that does not exist")

	// A Stream actor has no circles
	own := test.addCircle(t, test.userID)
	test.addMember(t, own, model.IdentifierTypeActivityPub, "https://remote.example/@mine", nil)

	asStream := test.collections
	asStream.actorType = model.ActorTypeStream
	members, err := asStream.Members(test.locator.CircleURL(test.userID, own))
	require.NoError(t, err)
	require.Empty(t, slices.Collect(members))
}

// TestOutbox2_crossesNetworkBoundary verifies that a domain delivers only to its own kind of network
func TestOutbox2_crossesNetworkBoundary(t *testing.T) {

	public := Outbox2{host: "https://example.com"}
	require.False(t, public.crossesNetworkBoundary("https://remote.example/@alice"))
	require.True(t, public.crossesNetworkBoundary("http://localhost:8080/@alice"))
	require.True(t, public.crossesNetworkBoundary("https://192.168.1.10/@alice"))
	require.False(t, public.crossesNetworkBoundary("mailto:alice@localhost"))

	local := Outbox2{host: "http://localhost:8080"}
	require.True(t, local.crossesNetworkBoundary("https://remote.example/@alice"), "a development server must never deliver to real followers")
	require.False(t, local.crossesNetworkBoundary("http://localhost:9090/@alice"))
	require.False(t, local.crossesNetworkBoundary("mailto:alice@remote.example"))
}

// TestOutbox2_localActor verifies that only a User or Stream URL on this server names a local actor
func TestOutbox2_localActor(t *testing.T) {

	service := Outbox2{host: "https://example.com"}
	actorID := primitive.NewObjectID()

	actorType, parsedID := service.localActor("https://example.com/@" + actorID.Hex())
	require.Equal(t, model.ActorTypeUser, actorType)
	require.Equal(t, actorID, parsedID)

	actorType, parsedID = service.localActor("https://example.com/" + actorID.Hex())
	require.Equal(t, model.ActorTypeStream, actorType)
	require.Equal(t, actorID, parsedID)

	actorType, _ = service.localActor("https://remote.example/@" + actorID.Hex())
	require.Empty(t, actorType)

	actorType, _ = service.localActor("")
	require.Empty(t, actorType)
}

// switchableEmailer is a followerEmailer that can be unconfigured, and can fail
type switchableEmailer struct {
	recordingEmailer
	configured bool
	failure    error
}

// IsConfigured reports whether email can be sent
func (emailer *switchableEmailer) IsConfigured() bool {
	return emailer.configured
}

// SendFollowerActivity records the send, or fails
func (emailer *switchableEmailer) SendFollowerActivity(session data.Session, follower *model.Follower, activity mapof.Any) error {

	if emailer.failure != nil {
		return emailer.failure
	}

	return emailer.recordingEmailer.SendFollowerActivity(session, follower, activity)
}

// queuedTasks records every task that a queue is asked to publish
type queuedTasks struct {
	mutex sync.Mutex
	tasks []queue.Task
}

// preProcessor records a copy of the task
func (recorder *queuedTasks) preProcessor(task *queue.Task) error {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.tasks = append(recorder.tasks, *task)
	return nil
}

// TestOutbox2_emailPublisher verifies that one signed task is queued per email recipient, and
// none on a domain that has no email configured
func TestOutbox2_emailPublisher(t *testing.T) {

	const actor = "https://example.com/@author"

	recorder := &queuedTasks{}
	taskQueue := queue.New(queue.WithPreProcessor(recorder.preProcessor), queue.WithBufferSize(8))
	emailer := &switchableEmailer{configured: true}
	service := Outbox2{host: "https://example.com", queue: taskQueue, domainEmail: emailer}

	publish := service.emailPublisher(actor)
	activity := map[string]any{"type": "Create", "object": map[string]any{"type": "Note", "content": "Hello"}}
	require.NoError(t, publish(actor+"/pub/outbox/1", "mailto:carol@remote.example", activity))

	require.Len(t, recorder.tasks, 1)
	task := recorder.tasks[0]
	require.Equal(t, TaskOutbox2Email, task.Name)
	require.Equal(t, actor+"/pub/outbox/1 mailto:carol@remote.example", task.Signature)
	require.Equal(t, "example.com", task.Arguments.GetString("hostname"))
	require.Equal(t, actor, task.Arguments.GetString("actor"))
	require.Equal(t, "carol@remote.example", task.Arguments.GetString("address"))
	require.Equal(t, "Create", task.Arguments.GetMap("activity").GetString("type"))

	// A domain with no email configured queues nothing, and says so once
	emailer.configured = false
	publish = service.emailPublisher(actor)
	require.NoError(t, publish(actor+"/pub/outbox/1", "mailto:dana@remote.example", activity))
	require.Len(t, recorder.tasks, 1)
}

// TestOutbox2_emailPublisher_ProfileUpdate verifies that a profile change reaches no email follower,
// because the email would show nothing
func TestOutbox2_emailPublisher_ProfileUpdate(t *testing.T) {

	const actor = "https://example.com/@author"

	recorder := &queuedTasks{}
	taskQueue := queue.New(queue.WithPreProcessor(recorder.preProcessor), queue.WithBufferSize(8))
	service := Outbox2{host: "https://example.com", queue: taskQueue, domainEmail: &switchableEmailer{configured: true}}

	// The same shape that User.SendProfileUpdate sends to its followers collection
	activity := map[string]any{
		"type":   "Update",
		"actor":  actor,
		"cc":     []any{actor + "/pub/followers"},
		"object": map[string]any{"id": actor, "type": "Person", "name": "Author"},
	}

	require.NoError(t, service.emailPublisher(actor)(actor+"#updates/1", "mailto:carol@remote.example", activity))
	require.Empty(t, recorder.tasks)
}

// TestOutbox2_SendEmail verifies that only a confirmed Follower of the sending actor is emailed
func TestOutbox2_SendEmail(t *testing.T) {

	test := newDeliveryTest(t)
	emailer := &switchableEmailer{configured: true}
	service := Outbox2{host: "https://example.com", followerService: &Follower{}, domainEmail: emailer}
	actor := test.locator.UserURL(test.userID)
	activity := mapof.Any{"type": "Create"}

	test.addFollower(t, model.FollowerTypeUser, test.userID, model.FollowerMethodEmail, "carol", model.FollowerStateActive)
	test.addFollower(t, model.FollowerTypeUser, test.userID, model.FollowerMethodEmail, "pending", model.FollowerStatePending)
	test.addFollower(t, model.FollowerTypeUser, primitive.NewObjectID(), model.FollowerMethodEmail, "elsewhere", model.FollowerStateActive)

	require.NoError(t, service.SendEmail(test.session, actor, "Carol@Remote.Example", activity))
	require.Equal(t, []string{"carol@remote.example"}, emailer.sent)

	// RULE: An unconfirmed Follower, another actor's Follower, and a stranger receive nothing
	require.NoError(t, service.SendEmail(test.session, actor, "pending@remote.example", activity))
	require.NoError(t, service.SendEmail(test.session, actor, "elsewhere@remote.example", activity))
	require.NoError(t, service.SendEmail(test.session, actor, "stranger@remote.example", activity))
	require.Len(t, emailer.sent, 1)

	// A failed send is returned, so that the task is retried
	emailer.failure = errors.New("smtp is down")
	require.Error(t, service.SendEmail(test.session, actor, "carol@remote.example", activity))
}

/******************************************
 * The Production Pipeline
 ******************************************/

// TestOutbox2_Pipeline_FanOut verifies who one fan-out through the real pipeline reaches, and that
// every exclusion Emissary supplies is applied: block rules, Floodgate, the network boundary,
// unconfirmed email, and collections that belong to another actor
func TestOutbox2_Pipeline_FanOut(t *testing.T) {

	// O5: the production pipeline had no test, so the wiring between Emissary's adapters and
	// hannibal's stages was proven only by reading it
	test := newPipelineTest(t, "https://example.com")
	other := primitive.NewObjectID()

	// The sender's followers
	test.addFollowerAt(t, model.FollowerMethodActivityPub, "https://remote.example/@alice", "", model.FollowerStateActive)
	test.addFollowerAt(t, model.FollowerMethodActivityPub, "https://remote.example/@bob", "", model.FollowerStateActive)
	test.addFollowerAt(t, model.FollowerMethodEmail, "carol@remote.example", "carol@remote.example", model.FollowerStateActive)
	test.addFollowerAt(t, model.FollowerMethodEmail, "pending@remote.example", "pending@remote.example", model.FollowerStatePending)
	test.addFollowerAt(t, model.FollowerMethodActivityPub, "https://blockeddomain.example/@eve", "", model.FollowerStateActive)
	test.addFollowerAt(t, model.FollowerMethodActivityPub, "https://paused.example/@pat", "", model.FollowerStateActive)
	test.addFollowerAt(t, model.FollowerMethodActivityPub, "http://localhost:9000/@lou", "", model.FollowerStateActive)

	// Someone else's follower, who must not be reached by naming their collection
	stranger := model.NewFollower()
	stranger.ParentType = model.FollowerTypeUser
	stranger.ParentID = other
	stranger.Method = model.FollowerMethodActivityPub
	stranger.StateID = model.FollowerStateActive
	stranger.Actor.ProfileURL = "https://remote.example/@stranger"
	require.NoError(t, test.session.Collection("Follower").Save(&stranger, "test"))

	// One of the sender's circles
	circleID := test.addCircle(t, test.userID)
	test.addMember(t, circleID, model.IdentifierTypeActivityPub, "https://circle.example/@cy", nil)

	// The sender's block rules, and a host that Floodgate holds back
	test.block(t, model.RuleTypeActor, "https://remote.example/@mallory")
	test.block(t, model.RuleTypeDomain, "blockeddomain.example")
	test.pause(t, "paused.example")

	activity := mapof.Any{
		"id":    test.actorURL + "/pub/outbox/1",
		"type":  "Create",
		"actor": test.actorURL,
		"to": []any{
			"https://www.w3.org/ns/activitystreams#Public",
			test.locator.FollowersURL(model.ActorTypeUser, test.userID),
			"https://remote.example/@bob",
			"https://remote.example/@mallory",
			test.locator.FollowersURL(model.ActorTypeUser, other),
		},
		"cc":     []any{test.locator.CircleURL(test.userID, circleID)},
		"bcc":    []any{"https://blind.example/@dana"},
		"object": mapof.Any{"id": test.actorURL + "/notes/1", "type": "Note", "content": "Hello"},
	}

	result := test.fanOut(activity)
	require.Equal(t, queue.ResultStatusSuccess, result.Status, result.Error)

	sends, emails := test.recipients()

	require.ElementsMatch(t, []string{
		"https://remote.example/@alice",
		"https://remote.example/@bob",
		"https://circle.example/@cy",
		"https://blind.example/@dana",
	}, sends, "Public, a blocked actor, a blocked domain, a paused host, another network, and another actor's followers receive nothing; bob receives one copy")

	require.Equal(t, []string{"carol@remote.example"}, emails, "only a confirmed email follower is emailed")

	// Every send is signed as the sender, and carries no blind recipients
	for _, task := range test.recorder.tasks {
		if task.Name == delivery.TaskSend {
			require.Equal(t, test.actorURL, task.Arguments.GetString("actor"))
			require.NotContains(t, task.Arguments.GetString("body"), "blind.example")
		}
	}
}

// TestOutbox2_Pipeline_FanOut_NotOurActor verifies that a fan-out for an actor this server cannot
// sign for ends without queueing anything
func TestOutbox2_Pipeline_FanOut_NotOurActor(t *testing.T) {

	test := newPipelineTest(t, "https://example.com")

	for name, actorURL := range map[string]string{
		"a remote actor":            "https://remote.example/@alice",
		"a User who does not exist": test.locator.UserURL(primitive.NewObjectID()),
	} {
		activity := mapof.Any{"type": "Create", "actor": actorURL, "to": "https://remote.example/@bob", "object": mapof.Any{"type": "Note", "content": "Hi"}}
		result := test.fanOut(activity)

		require.Equal(t, queue.ResultStatusFailure, result.Status, name)
	}

	sends, emails := test.recipients()
	require.Empty(t, sends)
	require.Empty(t, emails)
}

// TestOutbox2_Pipeline_FanOut_Unreachable verifies that a fan-out whose blocked hosts, block rules, or
// followers cannot be read is retried whole, and queues nothing
func TestOutbox2_Pipeline_FanOut_Unreachable(t *testing.T) {

	// test runs one fan-out with the provided failure and checks that it queued nothing
	test := func(name string, breakIt func(*pipelineTest) data.Session) {
		t.Run(name, func(t *testing.T) {

			fixture := newPipelineTest(t, "https://example.com")
			fixture.addFollowerAt(t, model.FollowerMethodActivityPub, "https://remote.example/@alice", "", model.FollowerStateActive)
			session := breakIt(&fixture)

			activity := mapof.Any{
				"type":   "Create",
				"actor":  fixture.actorURL,
				"to":     []any{fixture.locator.FollowersURL(model.ActorTypeUser, fixture.userID), "https://remote.example/@bob"},
				"object": mapof.Any{"type": "Note", "content": "Hi"},
			}

			result := fixture.run(session, queue.NewTask(delivery.TaskFanOut, mapof.Any{"activity": activity}))
			require.Equal(t, queue.ResultStatusError, result.Status)

			sends, emails := fixture.recipients()
			require.Empty(t, sends, "a partial list is never queued")
			require.Empty(t, emails)
		})
	}

	test("Floodgate", func(fixture *pipelineTest) data.Session {
		gate := floodgate.New(unreachableStore{})
		fixture.service.floodgate = func() floodgate.Gate { return gate }
		return fixture.session
	})

	test("block rules", func(fixture *pipelineTest) data.Session {
		return unreachableSession{Session: fixture.session, unreachable: map[string]bool{"Rule": true}}
	})

	test("followers", func(fixture *pipelineTest) data.Session {
		return unreachableSession{Session: fixture.session, unreachable: map[string]bool{"Follower": true}}
	})
}

// TestOutbox2_Pipeline_Send_CollectionOwner verifies that a send to a remote followers collection
// reaches the collection's owner, signed as the sender, unless the sender blocked the owner or
// Floodgate is holding back the owner's host
func TestOutbox2_Pipeline_Send_CollectionOwner(t *testing.T) {

	// test sends one reply to a remote author's followers collection, and returns what reached the author's inbox
	test := func(name string, prepare func(*pipelineTest, string), expected int) {
		t.Run(name, func(t *testing.T) {

			fixture := newPipelineTest(t, "http://localhost")
			remote := newInboxServer(t)

			// The remote author, and the followers collection that the reply names
			ownerURL := remote.URL + "/users/author"
			collectionURL := ownerURL + "/followers"

			fixture.cache(t, mapof.Any{"id": collectionURL, "type": "OrderedCollection", "totalItems": 3})
			fixture.cache(t, mapof.Any{"id": ownerURL, "type": "Person", "inbox": ownerURL + "/inbox", "followers": collectionURL})

			prepare(&fixture, ownerURL)

			reply := mapof.Any{
				"id":     fixture.actorURL + "/pub/outbox/1",
				"type":   "Create",
				"actor":  fixture.actorURL,
				"to":     []any{ownerURL},
				"cc":     []any{collectionURL},
				"object": mapof.Any{"type": "Note", "content": "Nice post"},
			}

			result := fixture.send(t, collectionURL, reply)
			require.Equal(t, queue.ResultStatusSuccess, result.Status, result.Error)

			deliveries := remote.received()
			require.Len(t, deliveries, expected)

			for _, request := range deliveries {
				require.Equal(t, "/users/author/inbox", request.URL.Path)
				require.Contains(t, request.Header.Get("Signature"), fixture.actorURL, "signed with the sender's own key")
			}
		})
	}

	test("the owner receives it", func(*pipelineTest, string) {}, 1)

	test("an owner the sender blocked receives nothing", func(fixture *pipelineTest, ownerURL string) {
		fixture.block(t, model.RuleTypeActor, ownerURL)
	}, 0)

	test("a host that Floodgate holds back receives nothing", func(fixture *pipelineTest, _ string) {
		fixture.pause(t, "127.0.0.1")
	}, 0)
}

// TestOutbox2_deliveryAllowed verifies the one-recipient test that a send task applies to the owner
// of a remote collection: the sender's block rules, and the network boundary
func TestOutbox2_deliveryAllowed(t *testing.T) {

	test := newPipelineTest(t, "https://example.com")
	test.block(t, model.RuleTypeActor, "https://remote.example/@mallory")
	test.block(t, model.RuleTypeDomain, "blockeddomain.example")

	allowed := func(recipientURL string) bool {
		return test.service.deliveryAllowed(test.session, test.userID, recipientURL)
	}

	require.True(t, allowed("https://remote.example/@alice"))
	require.False(t, allowed("https://remote.example/@mallory"), "an actor the sender blocked")
	require.False(t, allowed("https://blockeddomain.example/@eve"), "an actor on a domain the sender blocked")
	require.False(t, allowed("http://localhost:9000/@lou"), "an actor on another network")
	require.False(t, allowed(""), "a recipient that cannot be identified")

	// Another User's rules do not apply to this sender
	require.True(t, test.service.deliveryAllowed(test.session, primitive.NewObjectID(), "https://remote.example/@mallory"))
}

// TestOutbox2_blockedHosts verifies that a fan-out skips every host that Floodgate has marked
// dormant or paused, and no other
func TestOutbox2_blockedHosts(t *testing.T) {

	gate := floodgate.New(floodgate.NewMemoryStore())
	require.NoError(t, gate.SetPaused(context.Background(), "paused.example", true))

	service := Outbox2{floodgate: func() floodgate.Gate { return gate }}

	blocked, err := service.blockedHosts()
	require.NoError(t, err)
	require.True(t, blocked.Contains("paused.example"))
	require.False(t, blocked.Contains("remote.example"))

	// A Store that cannot answer fails the fan-out, rather than delivering to a dormant host
	broken := floodgate.New(unreachableStore{})
	service.floodgate = func() floodgate.Gate { return broken }

	_, err = service.blockedHosts()
	require.Error(t, err)
}

// TestOutbox2_remoteOptions verifies that every request gets a new Floodgate option of its own
func TestOutbox2_remoteOptions(t *testing.T) {

	calls := 0
	gate := floodgate.New(floodgate.NewMemoryStore())
	service := Outbox2{floodgate: func() floodgate.Gate { calls++; return gate }}

	require.Len(t, service.remoteOptions(), 1)
	require.Len(t, service.remoteOptions(), 1)
	require.Equal(t, 2, calls, "the Gate is read for each request, so a reload is seen at once")
}

// TestOutbox2_emailPublisher_QueueRefuses verifies that an email task the queue refuses fails the fan-out,
// so that it is retried
func TestOutbox2_emailPublisher_QueueRefuses(t *testing.T) {

	const actor = "https://example.com/@author"

	refusing := queue.New(queue.WithPreProcessor(func(*queue.Task) error { return errors.New("queue is full") }))
	service := Outbox2{host: "https://example.com", queue: refusing, domainEmail: &switchableEmailer{configured: true}}

	activity := map[string]any{"type": "Create", "object": map[string]any{"type": "Note", "content": "Hello"}}
	require.Error(t, service.emailPublisher(actor)(actor+"/pub/outbox/1", "mailto:carol@remote.example", activity))
}

// TestOutbox2_SendEmail_Unreachable verifies that a Follower who cannot be loaded fails the email
// task, so that it is retried, rather than being skipped as a stranger
func TestOutbox2_SendEmail_Unreachable(t *testing.T) {

	emailer := &switchableEmailer{configured: true}
	service := Outbox2{host: "https://example.com", followerService: &Follower{}, domainEmail: emailer}

	err := service.SendEmail(failingSession{}, "https://example.com/@"+primitive.NewObjectID().Hex(), "carol@remote.example", mapof.Any{"type": "Create"})
	require.Error(t, err)
	require.False(t, derp.IsNotFound(err))
	require.Empty(t, emailer.sent)
}

// TestDeliveryCollections_NotACollection verifies that a URL that is not one of this server's
// collections has no members
func TestDeliveryCollections_NotACollection(t *testing.T) {

	collections := deliveryCollections{host: "https://example.com", locator: &Locator{host: "https://example.com"}, actorType: model.ActorTypeUser, actorID: primitive.NewObjectID()}

	for _, collectionURL := range []string{"https://remote.example/@alice/followers", "https://example.com/@alice", "", "mailto:someone@example.com"} {
		members, err := collections.Members(collectionURL)
		require.NoError(t, err, collectionURL)
		require.Empty(t, slices.Collect(members), collectionURL)
	}
}

// TestDeliveryCollections_StopsEarly verifies that a stage that stops reading a collection stops
// its expansion, for followers and for a circle alike
func TestDeliveryCollections_StopsEarly(t *testing.T) {

	test := newDeliveryTest(t)

	test.addFollower(t, model.FollowerTypeUser, test.userID, model.FollowerMethodActivityPub, "alice", model.FollowerStateActive)
	test.addFollower(t, model.FollowerTypeUser, test.userID, model.FollowerMethodActivityPub, "bob", model.FollowerStateActive)

	circleID := test.addCircle(t, test.userID)
	test.addMember(t, circleID, model.IdentifierTypeActivityPub, "https://remote.example/@one", nil)
	test.addMember(t, circleID, model.IdentifierTypeActivityPub, "https://remote.example/@two", nil)

	for _, collectionURL := range []string{test.followers(model.ActorTypeUser, test.userID), test.locator.CircleURL(test.userID, circleID)} {

		members, err := test.collections.Members(collectionURL)
		require.NoError(t, err)

		count := 0

		for range members {
			count++
			break
		}

		require.Equal(t, 1, count, collectionURL)
	}
}

// TestDeliveryCollections_Unreachable verifies that a collection whose records cannot be read
// fails, rather than reading as empty
func TestDeliveryCollections_Unreachable(t *testing.T) {

	fixture := newDeliveryTest(t)
	circleID := fixture.addCircle(t, fixture.userID)

	// test reads one collection with the named collections unreachable, and requires an error
	test := func(name string, collectionURL string, unreachable string) {
		t.Run(name, func(t *testing.T) {

			collections := fixture.collections
			collections.session = unreachableSession{Session: fixture.session, unreachable: map[string]bool{unreachable: true}}

			_, err := collections.Members(collectionURL)
			require.Error(t, err)
		})
	}

	test("followers", fixture.followers(model.ActorTypeUser, fixture.userID), "Follower")
	test("a circle", fixture.locator.CircleURL(fixture.userID, circleID), "Circle")
	test("a circle's members", fixture.locator.CircleURL(fixture.userID, circleID), "Privilege")
}

// TestFollowerRecipient verifies the address each kind of Follower receives activities at
func TestFollowerRecipient(t *testing.T) {

	// test checks the address of one Follower
	test := func(name string, method string, stateID string, profileURL string, emailAddress string, expected string) {
		t.Run(name, func(t *testing.T) {
			follower := model.NewFollower()
			follower.Method = method
			follower.StateID = stateID
			follower.Actor.ProfileURL = profileURL
			follower.Actor.EmailAddress = emailAddress
			require.Equal(t, expected, followerRecipient(&follower))
		})
	}

	test("an ActivityPub follower, by actor", model.FollowerMethodActivityPub, model.FollowerStateActive, "https://remote.example/@alice", "", "https://remote.example/@alice")
	test("a confirmed email follower, by address", model.FollowerMethodEmail, model.FollowerStateActive, "", "carol@remote.example", "mailto:carol@remote.example")
	test("an unconfirmed email follower", model.FollowerMethodEmail, model.FollowerStatePending, "", "pending@remote.example", "")
	test("an email follower with no address", model.FollowerMethodEmail, model.FollowerStateActive, "https://remote.example/@nobody", "", "")
	test("a follower by an unknown method", "WEBSUB", model.FollowerStateActive, "https://remote.example/@feed", "feed@remote.example", "")
	test("a follower with no method", "", model.FollowerStateActive, "https://remote.example/@feed", "feed@remote.example", "")
}

// TestDeliveryCollections_memberRecipient verifies the address a circle member who has neither an
// actor nor an email address receives activities at
func TestDeliveryCollections_memberRecipient(t *testing.T) {

	collections := deliveryCollections{session: brokenSession{}, identityService: &Identity{}}

	privilege := model.NewPrivilege()
	privilege.IdentifierType = model.IdentifierTypeWebfinger
	privilege.IdentifierValue = "@someone@remote.example"

	require.Empty(t, collections.memberRecipient(&privilege), "a WebFinger handle is not an address")
}

/******************************************
 * Fuzz Tests
 ******************************************/

// FuzzDeliveryCollections_Members verifies that any recipient URL is read without panicking, that
// only the sender's own followers collection yields anyone, and that a URL that is not a
// collection yields no one
func FuzzDeliveryCollections_Members(f *testing.F) {

	const host = "https://example.com"

	locator := &Locator{host: host}
	senderID := primitive.NewObjectID()

	f.Add(locator.FollowersURL(model.ActorTypeUser, senderID))
	f.Add(locator.FollowersURL(model.ActorTypeStream, senderID))
	f.Add(locator.FollowersURL(model.ActorTypeUser, primitive.NewObjectID()))
	f.Add(locator.CircleURL(senderID, primitive.NewObjectID()))
	f.Add("followers:" + senderID.Hex())
	f.Add(host + "/@search_" + senderID.Hex() + "/pub/followers")
	f.Add(host + "//pub/followers")
	f.Add("https://remote.example/@alice/followers")
	f.Add("mailto:alice@example.com")
	f.Add("\xff\xfe")

	// Every Follower query answers with one follower, so a collection that is expanded yields someone
	follower := model.NewFollower()
	follower.Method = model.FollowerMethodActivityPub
	follower.Actor.ProfileURL = "https://remote.example/@alice"

	collections := deliveryCollections{
		session:          brokenSession{records: map[string][]data.Object{"Follower": {&follower}}},
		host:             host,
		locator:          locator,
		followerService:  &Follower{},
		circleService:    &Circle{},
		privilegeService: &Privilege{},
		identityService:  &Identity{},
		actorType:        model.ActorTypeUser,
		actorID:          senderID,
	}

	f.Fuzz(func(t *testing.T, collectionURL string) {

		isCollection := collections.IsCollection(collectionURL)

		members, err := collections.Members(collectionURL)
		require.NoError(t, err)

		if len(slices.Collect(members)) == 0 {
			return
		}

		// RULE: Only the sender's own followers collection is expanded
		require.True(t, isCollection, collectionURL)
		actorType, actorID := parseFollowersURI(host, collectionURL)
		require.Equal(t, model.ActorTypeUser, actorType, collectionURL)
		require.Equal(t, senderID, actorID, collectionURL)
	})
}

// FuzzOutbox2_crossesNetworkBoundary verifies that any recipient is checked without panicking, and
// that an email address never crosses the boundary
func FuzzOutbox2_crossesNetworkBoundary(f *testing.F) {

	f.Add("https://remote.example/@alice", false)
	f.Add("http://localhost:8080/@alice", true)
	f.Add("https://192.168.1.10/@alice", false)
	f.Add("mailto:alice@localhost", false)
	f.Add("MAILTO:alice@remote.example", true)
	f.Add("", true)
	f.Add("\xff", false)

	public := Outbox2{host: "https://example.com"}
	local := Outbox2{host: "http://localhost:8080"}

	f.Fuzz(func(t *testing.T, recipientURL string, isLocal bool) {

		service := public

		if isLocal {
			service = local
		}

		// Any other recipient is checked only for a panic; its answer is uri.IsLocalHostname's
		if !delivery.IsEmail(recipientURL) {
			_ = service.crossesNetworkBoundary(recipientURL)
			return
		}

		require.False(t, service.crossesNetworkBoundary(recipientURL), recipientURL)
	})
}

// FuzzOutbox2_localActor verifies that any actor URL is read without panicking, that it names a
// local actor only beneath this host, and that a local actor's own URL names it again
func FuzzOutbox2_localActor(f *testing.F) {

	const host = "https://example.com"

	actorID := primitive.NewObjectID()

	f.Add(host + "/@" + actorID.Hex())
	f.Add(host + "/" + actorID.Hex())
	f.Add(host + "/@search_" + actorID.Hex())
	f.Add("https://remote.example/@" + actorID.Hex())
	f.Add("followers:" + actorID.Hex())
	f.Add(host + "/@" + actorID.Hex() + "/pub/followers")
	f.Add("")

	service := Outbox2{host: host, locator: &Locator{host: host}}

	f.Fuzz(func(t *testing.T, actorURL string) {

		actorType, parsedID := service.localActor(actorURL)

		if actorType == "" {
			require.True(t, parsedID.IsZero(), actorURL)
			return
		}

		// A zero ID parses too. It names no actor, so the pipeline's signer refuses it first.
		require.Contains(t, []string{model.ActorTypeUser, model.ActorTypeStream, model.ActorTypeSearchQuery}, actorType, actorURL)
		require.True(t, strings.HasPrefix(actorURL, host+"/"), actorURL)

		// A User or Stream named by its own URL is found again
		if actorType != model.ActorTypeSearchQuery {
			roundTripType, roundTripID := service.localActor(service.locator.ActorURL(actorType, parsedID))
			require.Equal(t, actorType, roundTripType, actorURL)
			require.Equal(t, parsedID, roundTripID, actorURL)
		}
	})
}

// FuzzFollowerRecipient verifies that any Follower record yields its own actor, its own address,
// or nothing, and that an unconfirmed Follower is never emailed
func FuzzFollowerRecipient(f *testing.F) {

	f.Add(model.FollowerMethodActivityPub, model.FollowerStateActive, "https://remote.example/@alice", "")
	f.Add(model.FollowerMethodEmail, model.FollowerStateActive, "", "carol@remote.example")
	f.Add(model.FollowerMethodEmail, model.FollowerStatePending, "", "pending@remote.example")
	f.Add("", "", "", "")

	f.Fuzz(func(t *testing.T, method string, stateID string, profileURL string, emailAddress string) {

		follower := model.NewFollower()
		follower.Method = method
		follower.StateID = stateID
		follower.Actor.ProfileURL = profileURL
		follower.Actor.EmailAddress = emailAddress

		switch recipient := followerRecipient(&follower); recipient {

		case "":

		case profileURL:
			require.Equal(t, model.FollowerMethodActivityPub, method)

		case "mailto:" + emailAddress:
			require.Equal(t, model.FollowerMethodEmail, method)
			require.True(t, follower.IsActive())
			require.NotEmpty(t, emailAddress)

		default:
			require.Fail(t, "unexpected recipient", recipient)
		}
	})
}

// FuzzDeliveryCollections_memberRecipient verifies that any circle member yields the identifier
// they joined with, as an actor or an email address, or nothing
func FuzzDeliveryCollections_memberRecipient(f *testing.F) {

	f.Add(model.IdentifierTypeActivityPub, "https://remote.example/@member")
	f.Add(model.IdentifierTypeEmail, "member@remote.example")
	f.Add(model.IdentifierTypeWebfinger, "@member@remote.example")
	f.Add("", "")

	// The member's Identity cannot be found, as for a member who joined without one
	collections := deliveryCollections{session: brokenSession{}, identityService: &Identity{}}

	f.Fuzz(func(t *testing.T, identifierType string, identifierValue string) {

		privilege := model.NewPrivilege()
		privilege.IdentifierType = identifierType
		privilege.IdentifierValue = identifierValue

		switch recipient := collections.memberRecipient(&privilege); identifierType {

		case model.IdentifierTypeActivityPub:
			require.Equal(t, identifierValue, recipient)

		case model.IdentifierTypeEmail:
			require.Equal(t, "mailto:"+identifierValue, recipient)

		default:
			require.Empty(t, recipient)
		}
	})
}

// FuzzOutbox2_emailPublisher verifies that any activity is handled without panicking, and that an
// email task is queued exactly when the activity brings a post to show
func FuzzOutbox2_emailPublisher(f *testing.F) {

	f.Add(`{"type":"Create","object":{"type":"Note","content":"Hello"}}`, "mailto:carol@remote.example")
	f.Add(`{"type":"Update","object":{"type":"Person","name":"Author"}}`, "mailto:carol@remote.example")
	f.Add(`{"type":"Announce","object":{"type":"Create","object":{"name":"Song"}}}`, "carol@remote.example")
	f.Add(`{"type":"Delete","object":"https://example.com/1"}`, "")
	f.Add(`null`, "mailto:")

	const actor = "https://example.com/@author"

	f.Fuzz(func(t *testing.T, document string, address string) {

		activity := map[string]any{}

		if err := json.Unmarshal([]byte(document), &activity); err != nil {
			return
		}

		recorder := &queuedTasks{}
		taskQueue := queue.New(queue.WithPreProcessor(recorder.preProcessor), queue.WithBufferSize(4))
		service := Outbox2{host: "https://example.com", queue: taskQueue, domainEmail: &switchableEmailer{configured: true}}

		require.NoError(t, service.emailPublisher(actor)(actor+"/pub/outbox/1", address, activity))

		if _, isPost := followerPost(activity); !isPost {
			require.Empty(t, recorder.tasks)
			return
		}

		// The task names the bare address, with the one mailto: prefix that a recipient URL carries removed
		require.Len(t, recorder.tasks, 1)
		require.Equal(t, strings.TrimPrefix(address, "mailto:"), recorder.tasks[0].Arguments.GetString("address"))
	})
}
