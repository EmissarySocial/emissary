package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/postcommit"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/sender"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestPermissionsToHex pins the permission (de)serialization used to carry Outbox.Publish
// permissions across the "Outbox-Publish" task boundary. ObjectIDs don't survive task storage
// round-trips reliably, so Publish serializes them to hex and consumer.OutboxPublish re-parses
// them; a drift here would silently deliver activities to followers who lack view permission,
// or drop delivery to followers who have it. See POST-COMMIT-FEDERATION.md §5 / F2.
func TestPermissionsToHex(t *testing.T) {

	// Empty / nil serialize to an empty (non-nil) slice.
	require.Equal(t, []string{}, permissionsToHex(nil))
	require.Equal(t, []string{}, permissionsToHex(model.Permissions{}))

	a, err := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
	require.Nil(t, err)

	b, err := primitive.ObjectIDFromHex("5f2b8a9c1d4e6f0011223344")
	require.Nil(t, err)

	// Serialization preserves order and values.
	hexes := permissionsToHex(model.Permissions{a, b})
	require.Equal(t, []string{"507f1f77bcf86cd799439011", "5f2b8a9c1d4e6f0011223344"}, hexes)

	// Full round-trip back through the consumer-side parse (mirrors consumer.OutboxPublish).
	parsed := make(model.Permissions, 0)
	for _, hex := range hexes {
		id, err := primitive.ObjectIDFromHex(hex)
		require.Nil(t, err)
		parsed = append(parsed, id)
	}

	require.Equal(t, model.Permissions{a, b}, parsed)
}

// TestDeliver_StripsBlindRecipients confirms that no delivery task carries the bto/bcc lists
func TestDeliver_StripsBlindRecipients(t *testing.T) {

	userID := primitive.NewObjectID()

	// One active ActivityPub follower on the same (local) network as the sender
	follower := model.NewFollower()
	follower.FollowerID = primitive.NewObjectID()
	follower.ParentID = userID
	follower.ParentType = model.FollowerTypeUser
	follower.Method = model.FollowerMethodActivityPub
	follower.StateID = model.FollowerStateActive
	follower.Actor.ProfileURL = "http://localhost/@follower"
	follower.Actor.InboxURL = "http://localhost/@follower/pub/inbox"

	// A public activity that also names blind recipients on another network,
	// so the same-network rule skips them and no remote inbox lookup is needed
	activity := mapof.Any{
		vocab.PropertyID:    "http://localhost/@sender/pub/outbox/1",
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyActor: "http://localhost/@sender",
		vocab.PropertyTo:    []any{vocab.NamespaceActivityStreamsPublic},
		vocab.PropertyBTo:   []any{"https://blind.example/@bto"},
		vocab.PropertyBCC:   []any{"https://blind.example/@bcc"},
	}

	// Wire an Outbox to in-memory followers and rules, spooling every task it publishes
	ruleService, _ := newRuleService(&ruleStore{})
	spool := postcommit.NewTasks()
	session := deliverSession{
		context: postcommit.WithContext(context.Background(), spool),
		collections: map[string]data.Collection{
			"Follower": &followerCollection{records: []model.Follower{follower}},
			"Rule":     &ruleStore{},
		},
	}

	outboxService := Outbox{
		followerService: &Follower{},
		ruleService:     ruleService,
		domainEmail:     &DomainEmail{},
		host:            "http://localhost",
	}

	permissions := model.Permissions{model.MagicGroupIDAnonymous}
	err := outboxService.Deliver(session, model.FollowerTypeUser, userID, activity, permissions, nil, false)
	require.Nil(t, err)

	// Exactly one delivery reaches the follower, and it carries no blind-recipient lists
	tasks := spool.Drain()
	require.Len(t, tasks, 1)
	require.Equal(t, sender.OutboxSendToSingleRecipient, tasks[0].Name)
	require.Equal(t, follower.Actor.InboxURL, tasks[0].Arguments.GetString("inbox"))

	delivered := mapof.NewAny()
	require.NoError(t, json.Unmarshal([]byte(tasks[0].Arguments.GetString("body")), &delivered))
	require.NotContains(t, tasks[0].Arguments, "activity")
	require.NotContains(t, delivered, vocab.PropertyBTo)
	require.NotContains(t, delivered, vocab.PropertyBCC)
	require.Equal(t, activity[vocab.PropertyID], delivered[vocab.PropertyID])

	// The caller's activity keeps its addressing
	require.Contains(t, activity, vocab.PropertyBTo)
	require.Contains(t, activity, vocab.PropertyBCC)
}

// deliverSession is a data.Session that serves named in-memory collections within a fixed context
type deliverSession struct {
	context     context.Context
	collections map[string]data.Collection
}

// Collection implements the data.Session interface, returning the named in-memory collection
func (s deliverSession) Collection(name string) data.Collection { return s.collections[name] }

// Context implements the data.Session interface, returning the context that carries the task spool
func (s deliverSession) Context() context.Context { return s.context }

// Close implements the data.Session interface. The stub holds no resources to release.
func (s deliverSession) Close() {}
