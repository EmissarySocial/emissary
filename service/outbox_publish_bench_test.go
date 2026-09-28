package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// benchmarkFollowerCount is how many followers each benchmarked fan-out delivers to
const benchmarkFollowerCount = 100

// BenchmarkDeliver_Storage measures one fan-out plus the storage round trip of every delivery task it queues.
func BenchmarkDeliver_Storage(b *testing.B) {

	for fixture := newDeliverFixture(b, benchmarkFollowerCount, acceptInbox); b.Loop(); {
		for _, task := range fixture.deliver(b) {
			roundTripTask(b, task)
		}
	}
}

// BenchmarkDeliver_EndToEnd measures one fan-out, the storage round trip, and every signed delivery POST.
func BenchmarkDeliver_EndToEnd(b *testing.B) {

	for fixture := newDeliverFixture(b, benchmarkFollowerCount, acceptInbox); b.Loop(); {
		for _, task := range fixture.deliver(b) {
			stored := roundTripTask(b, task)
			result := fixture.sender.SendToSingleRecipient(stored.Arguments)
			require.Equal(b, queue.ResultStatusSuccess, result.Status)
		}
	}
}

// deliverFixture holds an Outbox wired to in-memory followers, and a Sender that signs with a real key
type deliverFixture struct {
	outbox   Outbox
	sender   sender.Sender
	userID   primitive.ObjectID
	activity mapof.Any
	session  deliverSession
	spool    *postcommit.Tasks
}

// newDeliverFixture builds the fixture: followers whose inboxes all point at a local test server
// that answers every delivery with the provided handler
func newDeliverFixture(t testing.TB, followerCount int, inbox http.HandlerFunc) *deliverFixture {

	t.Helper()

	// Silence per-delivery debug logging, which would otherwise interleave with benchmark output
	previousLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(previousLevel) })

	// Every delivery POSTs to this local inbox
	server := httptest.NewServer(inbox)
	t.Cleanup(server.Close)

	// Create the sending actor, with a real signing key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// The sender sits on a public host, so the same-network rule skips its own addressees
	// (the actor and its followers collection) and only the local followers are delivered to.
	actorURL := "https://sender.example/@sender"
	locator := benchmarkLocator{actor: sender.NewActor(actorURL, actorURL+"#main-key", privateKey)}

	senderQueue := queue.New()
	t.Cleanup(senderQueue.Stop)

	// Create the followers, all on the local network and all pointing at the test inbox
	userID := primitive.NewObjectID()
	followers := make([]model.Follower, 0, followerCount)

	for range followerCount {
		follower := model.NewFollower()
		follower.FollowerID = primitive.NewObjectID()
		follower.ParentID = userID
		follower.ParentType = model.FollowerTypeUser
		follower.Method = model.FollowerMethodActivityPub
		follower.StateID = model.FollowerStateActive
		follower.Actor.ProfileURL = "http://localhost/@follower-" + follower.FollowerID.Hex()
		follower.Actor.InboxURL = server.URL
		followers = append(followers, follower)
	}

	ruleService, _ := newRuleService(&ruleStore{})
	spool := postcommit.NewTasks()

	return &deliverFixture{
		outbox: Outbox{
			followerService: &Follower{},
			ruleService:     ruleService,
			domainEmail:     &DomainEmail{},
			host:            "http://localhost",
		},
		sender:   sender.New(locator, senderQueue, sender.AllowPrivateIPs(true)),
		userID:   userID,
		activity: benchmarkActivity(actorURL),
		spool:    spool,
		session: deliverSession{
			context: postcommit.WithContext(context.Background(), spool),
			collections: map[string]data.Collection{
				"Follower": &followerCollection{records: followers},
				"Rule":     &ruleStore{},
			},
		},
	}
}

// deliver runs one fan-out and returns the delivery tasks it queued
func (fixture *deliverFixture) deliver(t testing.TB) []queue.Task {

	permissions := model.Permissions{model.MagicGroupIDAnonymous}
	err := fixture.outbox.Deliver(fixture.session, model.FollowerTypeUser, fixture.userID, fixture.activity, permissions, nil, false)
	require.NoError(t, err)

	return fixture.spool.Drain()
}

// acceptInbox is an inbox handler that accepts every delivery
func acceptInbox(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}

// roundTripTask encodes a task to BSON and decodes it again, as turbine's storage does
func roundTripTask(t testing.TB, task queue.Task) queue.Task {

	encoded, err := bson.Marshal(task)
	require.NoError(t, err)

	result := queue.Task{}
	require.NoError(t, bson.Unmarshal(encoded, &result))

	return result
}

// benchmarkActivity returns a Mastodon-shaped public Create/Note of roughly 3.5 KB
func benchmarkActivity(actorURL string) mapof.Any {

	paragraph := "<p>Lorem ipsum dolor sit amet, <a href=\"https://example.com/tags/go\" class=\"mention hashtag\" rel=\"tag\">#<span>go</span></a> consectetur adipiscing elit.</p>"

	return mapof.Any{
		vocab.AtContext:     vocab.ContextTypeActivityStreams,
		vocab.PropertyID:    actorURL + "/pub/outbox/1",
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyActor: actorURL,
		vocab.PropertyTo:    []any{vocab.NamespaceActivityStreamsPublic},
		vocab.PropertyCC:    []any{actorURL + "/pub/followers"},
		vocab.PropertyObject: mapof.Any{
			vocab.PropertyID:           actorURL + "/1",
			vocab.PropertyType:         vocab.ObjectTypeNote,
			vocab.PropertyAttributedTo: actorURL,
			vocab.PropertySummary:      "Content warning: benchmarks",
			vocab.PropertyContent:      strings.Repeat(paragraph, 20),
			vocab.PropertyTag:          []any{mapof.Any{vocab.PropertyType: "Hashtag", vocab.PropertyName: "#go"}},
		},
	}
}

// benchmarkLocator resolves exactly one signing actor, and no recipients
type benchmarkLocator struct {
	actor sender.Actor
}

// Actor implements the sender.Locator interface, returning the one signing actor
func (locator benchmarkLocator) Actor(url string) (sender.Actor, error) {

	if url == locator.actor.ActorID() {
		return locator.actor, nil
	}

	return nil, derp.NotFound("benchmarkLocator.Actor", "Unknown actor", url)
}

// Recipient implements the sender.Locator interface. Fan-out here comes from Outbox.Deliver, not the Sender.
func (locator benchmarkLocator) Recipient(string) (iter.Seq[string], error) {
	return func(func(string) bool) {}, nil
}
