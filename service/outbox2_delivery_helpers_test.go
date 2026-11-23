package service

/******************************************
 * Outbox2 Delivery Test Fixture
 *
 * Wires an Outbox2 to hannibal's real delivery pipeline. The
 * database, block rules, signing keys, and cached remote
 * documents are real; the queue only records what is
 * published, and Floodgate keeps its records in memory.
 ******************************************/

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/floodgate"
	"github.com/benpate/hannibal/clients"
	"github.com/benpate/hannibal/delivery"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// pipelineTest holds an Outbox2 wired to hannibal's delivery pipeline, sending as one local User
type pipelineTest struct {
	deliveryTest
	service  *Outbox2
	recorder *queuedTasks
	gate     floodgate.Gate
	actorURL string
}

// newPipelineTest returns an Outbox2 on the provided host, sending as a new User who has a
// signing key. It skips when no database is reachable.
func newPipelineTest(t *testing.T, host string) pipelineTest {

	t.Helper()

	server := mongodb.NewServer(newDomainTestDatabase(t))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	session, err := server.Session(ctx)
	require.NoError(t, err)

	locator := &Locator{host: host}
	userID := primitive.NewObjectID()
	ruleService := NewRule()
	gate := floodgate.New(floodgate.NewMemoryStore())
	recorder := &queuedTasks{}

	// The sending User. EncryptionKey makes their signing key on first use.
	user := model.NewUser()
	user.UserID = userID
	user.ProfileURL = locator.UserURL(userID)
	require.NoError(t, session.Collection("User").Save(&user, "test"))

	// The app's client reads remote documents from the cache in this database, never the network
	activityService := NewActivityStream()
	activityService.hostname = "localhost"
	activityService.carpool = clients.NewCarpool()
	activityService.ruleService = &ruleService
	activityService.getCommonDatabase = func() data.Server { return server }
	activityService.newSession = func(time.Duration) (data.Session, context.CancelFunc, error) {
		return session, func() {}, nil
	}

	service := &Outbox2{
		activityService:  &activityService,
		followerService:  &Follower{},
		circleService:    &Circle{},
		privilegeService: &Privilege{},
		identityService:  &Identity{},
		ruleService:      &ruleService,
		domainEmail:      &switchableEmailer{configured: true},
		locator:          locator,
		queue:            queue.New(queue.WithPreProcessor(recorder.preProcessor), queue.WithBufferSize(64)),
		floodgate:        func() floodgate.Gate { return gate },
		host:             host,
		getSendLocator: func(session data.Session) SendLocator {
			return SendLocator{
				userService:          &User{},
				encryptionKeyService: &EncryptionKey{},
				locatorService:       locator,
				host:                 host,
				session:              session,
			}
		},
	}

	return pipelineTest{
		deliveryTest: deliveryTest{
			session: session,
			locator: locator,
			userID:  userID,
		},
		service:  service,
		recorder: recorder,
		gate:     gate,
		actorURL: locator.UserURL(userID),
	}
}

// run runs one task through the consumer that consumer.Delivery builds for the sending User
func (test pipelineTest) run(session data.Session, task queue.Task) queue.Result {
	return delivery.Consumer(test.service.Pipeline(session, test.actorURL, true)).Run(task)
}

// fanOut runs one fan-out of the activity, on the test's own session
func (test pipelineTest) fanOut(activity mapof.Any) queue.Result {
	return test.run(test.session, queue.NewTask(delivery.TaskFanOut, mapof.Any{"actor": test.actorURL, "activity": activity}))
}

// send runs one send task that delivers the activity to the recipient
func (test pipelineTest) send(t *testing.T, recipientURL string, activity mapof.Any) queue.Result {

	t.Helper()

	body, err := json.Marshal(activity)
	require.NoError(t, err)

	return test.run(test.session, queue.NewTask(delivery.TaskSend, mapof.Any{
		"actor":     test.actorURL,
		"recipient": recipientURL,
		"body":      string(body),
		"queuedAt":  time.Now().Unix(),
	}))
}

// recipients returns the "recipient" argument of every send task queued, and the "address" of every email task
func (test pipelineTest) recipients() (sends []string, emails []string) {

	test.recorder.mutex.Lock()
	defer test.recorder.mutex.Unlock()

	sends, emails = make([]string, 0), make([]string, 0)

	for _, task := range test.recorder.tasks {

		switch task.Name {

		case delivery.TaskSend:
			sends = append(sends, task.Arguments.GetString("recipient"))

		case TaskOutbox2Email:
			emails = append(emails, task.Arguments.GetString("address"))
		}
	}

	return sends, emails
}

// addFollowerAt saves one Follower of the sending User, known by the provided profile URL and address
func (test pipelineTest) addFollowerAt(t *testing.T, method string, profileURL string, emailAddress string, stateID string) {

	t.Helper()

	follower := model.NewFollower()
	follower.ParentType = model.FollowerTypeUser
	follower.ParentID = test.userID
	follower.Method = method
	follower.StateID = stateID
	follower.Actor.ProfileURL = profileURL
	follower.Actor.EmailAddress = emailAddress

	require.NoError(t, test.session.Collection("Follower").Save(&follower, "test"))
}

// block saves one of the sending User's own block rules
func (test pipelineTest) block(t *testing.T, ruleType string, trigger string) {

	t.Helper()

	rule := model.NewRule()
	rule.UserID = test.userID
	rule.Type = ruleType
	rule.Action = model.RuleActionBlock
	rule.Trigger = trigger
	rule.MatchKey = model.RuleMatchKey(ruleType, trigger)

	require.NoError(t, test.session.Collection("Rule").Save(&rule, "test"))
}

// pause tells Floodgate to stop every delivery to a host
func (test pipelineTest) pause(t *testing.T, host string) {
	t.Helper()
	require.NoError(t, test.gate.SetPaused(context.Background(), host, true))
}

// cache stores a remote document in the app's cache, so the client serves it without a fetch
func (test pipelineTest) cache(t *testing.T, document mapof.Any) {

	t.Helper()

	value := ascache.NewValue()
	value.URLs = append(value.URLs, document.GetString("id"))
	value.Object = document
	value.Received = time.Now().Unix()

	require.NoError(t, test.session.Collection("Document").Save(&value, "test"))
}

/******************************************
 * Remote Server
 ******************************************/

// inboxServer is a remote server that counts the deliveries its inboxes receive
type inboxServer struct {
	*httptest.Server
	mutex      sync.Mutex
	deliveries []*http.Request
}

// newInboxServer returns a remote server that accepts every POST, and stops when the test ends
func newInboxServer(t *testing.T) *inboxServer {

	t.Helper()

	result := &inboxServer{}

	result.Server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		if request.Method == http.MethodPost {
			result.mutex.Lock()
			result.deliveries = append(result.deliveries, request)
			result.mutex.Unlock()
		}

		writer.WriteHeader(http.StatusAccepted)
	}))

	t.Cleanup(result.Close)

	return result
}

// received returns a copy of the deliveries received so far
func (server *inboxServer) received() []*http.Request {
	server.mutex.Lock()
	defer server.mutex.Unlock()
	return append([]*http.Request{}, server.deliveries...)
}

/******************************************
 * Failing Dependencies
 ******************************************/

// unreachableStore is a Floodgate Store that cannot list the hosts it has blocked
type unreachableStore struct {
	floodgate.Store
}

// RangeBlocked fails, as a database that did not answer would
func (unreachableStore) RangeBlocked(context.Context) (iter.Seq[floodgate.Host], error) {
	return nil, derp.Internal("service.unreachableStore.RangeBlocked", "Synthetic failure")
}

// unreachableSession wraps a session, and fails every read from the named collections
type unreachableSession struct {
	data.Session
	unreachable map[string]bool
}

// Collection returns the named collection, or one whose every read fails
func (session unreachableSession) Collection(name string) data.Collection {

	if session.unreachable[name] {
		return unreachableCollection{}
	}

	return session.Session.Collection(name)
}

// unreachableCollection is a data.Collection whose every read fails with an internal error
type unreachableCollection struct {
	data.Collection
}

// Iterator fails with an internal error
func (unreachableCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("service.unreachableCollection.Iterator", "Synthetic failure")
}

// Query fails with an internal error
func (unreachableCollection) Query(any, exp.Expression, ...option.Option) error {
	return derp.Internal("service.unreachableCollection.Query", "Synthetic failure")
}

// Load fails with an internal error
func (unreachableCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.Internal("service.unreachableCollection.Load", "Synthetic failure")
}
