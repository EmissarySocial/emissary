package service

import (
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
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
func (emailer *switchableEmailer) SendFollowerActivity(follower *model.Follower, activity mapof.Any) error {

	if emailer.failure != nil {
		return emailer.failure
	}

	return emailer.recordingEmailer.SendFollowerActivity(follower, activity)
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

// TestHasEmailContent verifies that only an activity that brings an embedded post with something to show is emailed
func TestHasEmailContent(t *testing.T) {

	note := map[string]any{"type": "Note", "content": "Hello"}

	// test checks one activity
	test := func(name string, activity map[string]any, expected bool) {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, hasEmailContent(activity))
		})
	}

	test("a Create of a Note", map[string]any{"type": "Create", "object": note}, true)
	test("an Update of an Article with only a name", map[string]any{"type": "Update", "object": map[string]any{"type": "Article", "name": "Title"}}, true)
	test("an Announce of a Note", map[string]any{"type": "Announce", "object": note}, true)
	test("an object as mapof.Any", map[string]any{"type": "Create", "object": mapof.Any{"type": "Note", "content": "Hello"}}, true)

	test("a Create of an empty Note", map[string]any{"type": "Create", "object": map[string]any{"type": "Note"}}, false)
	test("a Create of an object named by its URL", map[string]any{"type": "Create", "object": "https://remote.example/notes/1"}, false)
	test("a Create with no object", map[string]any{"type": "Create"}, false)
	test("an Update of a Person", map[string]any{"type": "Update", "object": map[string]any{"type": "Person", "name": "Author"}}, false)
	test("an Update of a Service", map[string]any{"type": "Update", "object": map[string]any{"type": "Service", "name": "Stream"}}, false)
	test("a Delete", map[string]any{"type": "Delete", "object": map[string]any{"type": "Tombstone", "name": "Gone"}}, false)
	test("an Announce of a Delete", map[string]any{"type": "Announce", "object": map[string]any{"type": "Delete", "object": "https://remote.example/notes/1"}}, false)
	test("an Announce of a Create of a Note", map[string]any{"type": "Announce", "object": map[string]any{"type": "Create", "object": note}}, true)
	test("an Announce of a Create of a URL", map[string]any{"type": "Announce", "object": map[string]any{"type": "Create", "object": "https://remote.example/notes/1"}}, false)
	test("an Undo", map[string]any{"type": "Undo", "object": note}, false)
	test("an Accept", map[string]any{"type": "Accept", "object": map[string]any{"type": "Follow"}}, false)
	test("a Like", map[string]any{"type": "Like", "object": note}, false)
	test("no type", map[string]any{"object": note}, false)
	test("an empty activity", map[string]any{}, false)
	test("a nil activity", nil, false)
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
