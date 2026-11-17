package service

import (
	"context"
	"crypto"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	querysync "github.com/EmissarySocial/emissary/queries/sync"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/corpus"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/******************************************
 * Corpus Tests
 *
 * These publish every document in hannibal's corpus package
 * through Outbox2 against a real MongoDB, then run the queued
 * task through the delivery pipeline as a queue worker would
 * receive it. They skip when no database is reachable.
 ******************************************/

// corpusBob is an actor that the fediverse examples name directly, and who also follows the author
const corpusBob = "https://mastodon.example.org/users/bob"

// corpusAlice follows the author over ActivityPub, and is named by no example
const corpusAlice = "https://follower.example.org/users/alice"

// corpusDana follows the author by email
const corpusDana = "mailto:dana@example.org"

// outbox2CorpusTest holds an Outbox2 service for one User on example.com, who has three followers
type outbox2CorpusTest struct {
	service     *Outbox2
	session     data.Session
	spool       *postcommit.Tasks
	userID      primitive.ObjectID
	world       corpus.World
	collections deliveryCollections
}

// newOutbox2CorpusTest returns an Outbox2 service over a throwaway database, for a User followed
// by two remote actors and one confirmed email address
func newOutbox2CorpusTest(t *testing.T) outbox2CorpusTest {

	t.Helper()

	database := newDomainTestDatabase(t)
	require.NoError(t, querysync.OutboxItem(context.Background(), database))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Tasks published in this session wait in the spool, as they do inside a transaction
	spool := postcommit.NewTasks()

	session, err := mongodb.NewServer(database).Session(postcommit.WithContext(ctx, spool))
	require.NoError(t, err)

	locator := &Locator{host: "https://example.com"}
	userID := primitive.NewObjectID()

	test := outbox2CorpusTest{
		service: &Outbox2{locator: locator, host: "https://example.com", inboxService: &Inbox{}},
		session: session,
		spool:   spool,
		userID:  userID,
		collections: deliveryCollections{
			session:          session,
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

	test.addFollower(t, model.FollowerMethodActivityPub, corpusAlice, "")
	test.addFollower(t, model.FollowerMethodActivityPub, corpusBob, "")
	test.addFollower(t, model.FollowerMethodEmail, "", "dana@example.org")

	// The members are read back through the adapter, so the examples expect the order it returns
	followersURL := locator.FollowersURL(model.ActorTypeUser, userID)
	members, err := test.collections.Members(followersURL)
	require.NoError(t, err)

	test.world = corpus.World{
		Actor:     locator.UserURL(userID),
		Followers: followersURL,
		Members:   slices.Collect(members),
	}

	require.ElementsMatch(t, []string{corpusAlice, corpusBob, corpusDana}, test.world.Members)

	return test
}

// addFollower saves one active Follower of the test User
func (test outbox2CorpusTest) addFollower(t *testing.T, method string, profileURL string, emailAddress string) {

	t.Helper()

	follower := model.NewFollower()
	follower.ParentType = model.FollowerTypeUser
	follower.ParentID = test.userID
	follower.Method = method
	follower.StateID = model.FollowerStateActive
	follower.Actor.ProfileURL = profileURL
	follower.Actor.EmailAddress = emailAddress

	require.NoError(t, test.session.Collection("Follower").Save(&follower, "test"))
}

// TestOutbox2_Corpus_Fediverse verifies that every fediverse example is stored whole, served
// without its blind recipients, and delivered to the recipients recorded for it
func TestOutbox2_Corpus_Fediverse(t *testing.T) {

	test := newOutbox2CorpusTest(t)

	examples, err := corpus.Fediverse(test.world)
	require.NoError(t, err)

	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {

			outcome := test.publish(t, example)

			// The task names the sending actor, because the worker builds its pipeline from it
			require.Equal(t, test.world.Actor, outcome.task.Arguments.GetString("actor"))

			addresses, emails := splitCorpusEmails(example.Recipients)
			require.Equal(t, addresses, outcome.recipients, "send tasks")
			require.Equal(t, emails, outcome.emails, "email recipients")

			// RULE: Nothing from the blind.example.org host is ever served or sent
			require.NotContains(t, outcome.served, "blind.example.org")

			for _, body := range outcome.bodies {
				require.NotContains(t, body, "blind.example.org")
			}
		})
	}
}

// TestOutbox2_Corpus_W3C verifies that every document the W3C publishes survives the outbox:
// it is stored whole, served without blind recipients, and delivered to whoever it names
func TestOutbox2_Corpus_W3C(t *testing.T) {

	test := newOutbox2CorpusTest(t)

	published, err := corpus.W3C()
	require.NoError(t, err)

	rejected, err := corpus.Rejected()
	require.NoError(t, err)

	// Several documents share one id, and an id is published once
	seen := make(map[string]bool)

	for _, example := range append(published, rejected...) {
		t.Run(example.Name, func(t *testing.T) {

			document := decodeCorpusJSON(t, example.JSON)

			// RULE: A document that names no actor is refused, and leaves nothing behind. It is
			// then published as the test's own actor, so that it still goes through the outbox.
			if corpusActorID(document["actor"]) == "" {
				test.publishRefused(t, example)
				document["actor"] = test.world.Actor
				example.JSON = []byte(marshalCorpusJSON(t, document))
			}

			if id, _ := document["id"].(string); id != "" {

				if seen[id] {
					test.publishAgain(t, example, id)
					return
				}

				seen[id] = true
			}

			outcome := test.publish(t, example)

			addresses, emails := splitCorpusEmails(namedCorpusRecipients(document))
			require.Equal(t, addresses, outcome.recipients, "send tasks")
			require.Equal(t, emails, outcome.emails, "email recipients")
		})
	}
}

// TestOutbox2_Corpus_PutUserActivity verifies the client-to-server door: a public example is
// refused, and every other one is stored under an id that the server minted
func TestOutbox2_Corpus_PutUserActivity(t *testing.T) {

	test := newOutbox2CorpusTest(t)

	examples, err := corpus.Fediverse(test.world)
	require.NoError(t, err)

	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {

			document := decodeCorpusJSON(t, example.JSON)
			originalID, _ := document["id"].(string)

			activityURL, err := test.service.PutUserActivity(test.session, test.userID, streams.NewDocument(mapof.Any(document)))
			tasks := test.spool.Drain()

			// RULE: A public post is a Stream, and is not accepted here
			if namesPublic(decodeCorpusJSON(t, example.JSON)) {
				require.True(t, derp.IsNotImplemented(err), "%v", err)
				require.Empty(t, tasks)
				return
			}

			require.NoError(t, err)
			require.Len(t, tasks, 1)

			// RULE: The server assigns the id, whatever the client sent
			require.True(t, strings.HasPrefix(activityURL, test.world.Actor+"/pub/outbox/"), activityURL)
			require.NotEqual(t, originalID, activityURL)

			item := model.NewOutboxItem()
			require.NoError(t, test.session.Collection("OutboxItem").Load(exp.Equal("url", activityURL), &item))
			require.Equal(t, activityURL, item.Activity.GetString("id"))
			require.NotContains(t, item.Activity, "@id", "a client cannot name its own activity in the JSON-LD spelling either")
		})
	}
}

/******************************************
 * Corpus Helpers
 ******************************************/

// corpusOutcome is what publishing one example stored, served, queued and delivered
type corpusOutcome struct {
	task       queue.Task
	served     string
	recipients []string
	emails     []string
	bodies     []string
}

// publish sends one example through Outbox2.Publish and then through the delivery fan-out, and
// checks everything that holds for any document
func (test outbox2CorpusTest) publish(t *testing.T, example corpus.Example) corpusOutcome {

	t.Helper()

	document := decodeCorpusJSON(t, example.JSON)

	activityURL, err := test.service.Publish(test.session, model.ActorTypeUser, test.userID, streams.NewDocument(mapof.Any(document)), model.NewPermissions())
	require.NoError(t, err)
	require.NotEmpty(t, activityURL)

	// The document as it should be stored: unchanged, apart from an id when it had none
	expected := decodeCorpusJSON(t, example.JSON)
	expected["id"] = activityURL
	expectedJSON := marshalCorpusJSON(t, expected)
	strippedJSON := marshalCorpusJSON(t, withoutBlindCorpusKeys(decodeCorpusJSON(t, []byte(expectedJSON))))

	// Read the row back, so that everything below sees the types MongoDB returns
	item := model.NewOutboxItem()
	require.NoError(t, test.session.Collection("OutboxItem").Load(exp.Equal("url", activityURL), &item))
	require.JSONEq(t, expectedJSON, marshalCorpusJSON(t, item.Activity), "the stored activity")
	require.Equal(t, test.userID, item.ActorID)

	// The outbox collection serves it without blind recipients, and the row keeps them
	served := marshalCorpusJSON(t, item.GetJSONLD())
	require.JSONEq(t, strippedJSON, served, "the served activity")
	require.JSONEq(t, expectedJSON, marshalCorpusJSON(t, item.Activity), "serving changed the stored activity")

	// One task fans the activity out, and carries it with its blind recipients
	tasks := test.spool.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, delivery.TaskFanOut, tasks[0].Name)

	// A worker receives the task from MongoDB, so run it as MongoDB returns it
	task := throughBSON(t, tasks[0])
	require.JSONEq(t, expectedJSON, marshalCorpusJSON(t, task.Arguments["activity"]), "the queued activity")

	outcome := corpusOutcome{task: task, served: served, emails: make([]string, 0)}

	recorder := &corpusTaskRecorder{}
	client := &countingClient{}

	pipeline := delivery.New(
		queue.New(queue.WithPreProcessor(recorder.preProcessor), queue.WithBufferSize(1024)),
		client,
		func(string) (delivery.Signer, error) { return corpusSigner{}, nil },
		delivery.WithCollections(test.collections),
		delivery.WithEmail(func(_ string, address string, activity map[string]any) error {
			outcome.emails = append(outcome.emails, address)
			require.JSONEq(t, strippedJSON, marshalCorpusJSON(t, activity), "the emailed activity")
			return nil
		}),
	)

	result := pipeline.FanOut(task.Arguments)
	require.Equal(t, queue.ResultStatusSuccess, result.Status, "%v", result.Error)
	require.Zero(t, client.loads, "a fan-out loads nothing from the network")

	outcome.recipients = make([]string, 0, len(recorder.tasks))

	for _, sendTask := range recorder.tasks {
		body := sendTask.Arguments.GetString("body")
		require.JSONEq(t, strippedJSON, body, "the delivered activity")
		outcome.recipients = append(outcome.recipients, sendTask.Arguments.GetString("recipient"))
		outcome.bodies = append(outcome.bodies, body)
	}

	return outcome
}

// publishRefused verifies that a document with no actor is not stored and queues nothing
func (test outbox2CorpusTest) publishRefused(t *testing.T, example corpus.Example) {

	t.Helper()

	before, err := test.session.Collection("OutboxItem").Count(exp.All())
	require.NoError(t, err)

	document := decodeCorpusJSON(t, example.JSON)

	activityURL, err := test.service.Publish(test.session, model.ActorTypeUser, test.userID, streams.NewDocument(mapof.Any(document)), model.NewPermissions())
	require.Error(t, err)
	require.True(t, derp.IsInternalServerError(err))
	require.Empty(t, activityURL)

	after, err := test.session.Collection("OutboxItem").Count(exp.All())
	require.NoError(t, err)
	require.Equal(t, before, after, "a refused document is not stored")
	require.Empty(t, test.spool.Drain(), "a refused document queues nothing")
}

// corpusActorID is this test's own reading of an `actor` value: a URL, an object with an id,
// or the first of a list
func corpusActorID(value any) string {

	switch typed := value.(type) {

	case string:
		return typed

	case map[string]any:
		id, _ := typed["id"].(string)
		return id

	case []any:
		if len(typed) > 0 {
			return corpusActorID(typed[0])
		}
	}

	return ""
}

// publishAgain verifies that an id which is already in the outbox is stored and sent once
func (test outbox2CorpusTest) publishAgain(t *testing.T, example corpus.Example, id string) {

	t.Helper()

	before, err := test.session.Collection("OutboxItem").Count(exp.All())
	require.NoError(t, err)

	activityURL, err := test.service.Publish(test.session, model.ActorTypeUser, test.userID, streams.NewDocument(mapof.Any(decodeCorpusJSON(t, example.JSON))), model.NewPermissions())
	require.NoError(t, err)
	require.Equal(t, id, activityURL)
	require.Empty(t, test.spool.Drain(), "a repeated id queues nothing")

	after, err := test.session.Collection("OutboxItem").Count(exp.All())
	require.NoError(t, err)
	require.Equal(t, before, after, "a repeated id stores nothing")
}

// throughBSON returns a task as a worker reads it back from the queue's MongoDB collection
func throughBSON(t *testing.T, task queue.Task) queue.Task {

	t.Helper()

	encoded, err := bson.Marshal(task)
	require.NoError(t, err)

	result := queue.Task{}
	require.NoError(t, bson.Unmarshal(encoded, &result))
	return result
}

// decodeCorpusJSON returns a document decoded as plain JSON
func decodeCorpusJSON(t *testing.T, document []byte) map[string]any {

	t.Helper()

	result := make(map[string]any)
	require.NoError(t, json.Unmarshal(document, &result))
	return result
}

// marshalCorpusJSON returns a value as JSON text
func marshalCorpusJSON(t *testing.T, value any) string {

	t.Helper()

	result, err := json.Marshal(value)
	require.NoError(t, err)
	return string(result)
}

// withoutBlindCorpusKeys deletes `bto` and `bcc` from every object in a decoded JSON value, and
// returns it. It is the simplest strip there is, written apart from the one under test.
func withoutBlindCorpusKeys(value any) any {

	switch typed := value.(type) {

	case map[string]any:
		delete(typed, "bto")
		delete(typed, "bcc")
		for _, item := range typed {
			withoutBlindCorpusKeys(item)
		}

	case []any:
		for _, item := range typed {
			withoutBlindCorpusKeys(item)
		}
	}

	return value
}

// namedCorpusRecipients reads the addressing of a decoded JSON document the plain way: every id
// in `to`, `cc`, `bto` and `bcc`, once each, without Public and without anything that has no host
func namedCorpusRecipients(document map[string]any) []string {

	result := make([]string, 0)

	for _, address := range corpusAddresses(document) {
		if isCorpusDeliverable(address) && !slices.Contains(result, address) {
			result = append(result, address)
		}
	}

	return result
}

// namesPublic returns TRUE if a decoded JSON document names Public anywhere in its addressing
func namesPublic(document map[string]any) bool {

	for _, address := range corpusAddresses(document) {
		if isCorpusPublic(address) {
			return true
		}
	}

	return false
}

// corpusAddresses returns every id in a document's `to`, `cc`, `bto` and `bcc`, in order
func corpusAddresses(document map[string]any) []string {

	result := make([]string, 0)

	var read func(value any)

	read = func(value any) {

		switch typed := value.(type) {

		case string:
			result = append(result, typed)

		case map[string]any:
			if id, isString := typed["id"].(string); isString {
				result = append(result, id)
			}

		case []any:
			for _, item := range typed {
				read(item)
			}
		}
	}

	for _, field := range []string{"to", "cc", "bto", "bcc"} {
		read(document[field])
	}

	return result
}

// isCorpusPublic returns TRUE if an address is one of the three spellings of Public
func isCorpusPublic(address string) bool {

	switch address {
	case "https://www.w3.org/ns/activitystreams#Public", "as:Public", "Public":
		return true
	}

	return false
}

// isCorpusDeliverable returns TRUE if an address is an email address or a URL with a host, and is not Public
func isCorpusDeliverable(address string) bool {

	if isCorpusPublic(address) {
		return false
	}

	if strings.HasPrefix(address, "mailto:") {
		return true
	}

	parsed, err := url.Parse(address)

	if (err != nil) || (parsed == nil) {
		return false
	}

	return parsed.Host != ""
}

// splitCorpusEmails separates a list of recipients into the URLs that get a send task and the
// mailto: URLs that are handed to the application
func splitCorpusEmails(recipients []string) ([]string, []string) {

	addresses := make([]string, 0, len(recipients))
	emails := make([]string, 0)

	for _, recipient := range recipients {
		if strings.HasPrefix(recipient, "mailto:") {
			emails = append(emails, recipient)
			continue
		}
		addresses = append(addresses, recipient)
	}

	return addresses, emails
}

// corpusTaskRecorder records every task that a queue is asked to publish
type corpusTaskRecorder struct {
	mutex sync.Mutex
	tasks []queue.Task
}

// preProcessor records a copy of the task
func (recorder *corpusTaskRecorder) preProcessor(task *queue.Task) error {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.tasks = append(recorder.tasks, *task)
	return nil
}

// corpusSigner is a delivery.Signer that a fan-out asks for and never signs with
type corpusSigner struct{}

// ActorID returns no actor, because the fan-out does not read it
func (corpusSigner) ActorID() string {
	return ""
}

// PrivateKey returns no key, because the fan-out does not sign
func (corpusSigner) PrivateKey() (string, crypto.PrivateKey) {
	return "", nil
}
