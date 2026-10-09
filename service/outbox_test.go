package service

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// outboxExportCollection is an in-memory collection for export query tests.
type outboxExportCollection struct {
	records []model.OutboxMessage
}

// Context implements the data.Collection interface.
func (c *outboxExportCollection) Context() context.Context { return context.Background() }

// Count implements the data.Collection interface.
func (c *outboxExportCollection) Count(exp.Expression, ...option.Option) (int64, error) {
	return 0, derp.Internal("test", "unused")
}

// Query returns IDs for records matching the export owner criteria.
func (c *outboxExportCollection) Query(target any, criteria exp.Expression, _ ...option.Option) error {
	ids, ok := target.(*[]model.IDOnly)
	if !ok {
		return derp.Internal("test", "unexpected query target type")
	}

	for _, record := range c.records {
		if criteria.Match(func(predicate exp.Predicate) bool {
			switch predicate.Field {
			case "actorId":
				ownerID, ok := predicate.Value.(primitive.ObjectID)
				return ok && ownerID == record.ActorID
			case "deleteDate":
				deleteDate, ok := predicate.Value.(int)
				return ok && deleteDate == int(record.DeleteDate)
			default:
				return false
			}
		}) {
			*ids = append(*ids, model.IDOnly{ID: record.OutboxMessageID})
		}
	}

	return nil
}

// Iterator implements the data.Collection interface.
func (c *outboxExportCollection) Iterator(exp.Expression, ...option.Option) (data.Iterator, error) {
	return nil, derp.Internal("test", "unused")
}

// Load implements the data.Collection interface.
func (c *outboxExportCollection) Load(exp.Expression, data.Object, ...option.Option) error {
	return derp.Internal("test", "unused")
}

// Save implements the data.Collection interface.
func (c *outboxExportCollection) Save(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// Delete implements the data.Collection interface.
func (c *outboxExportCollection) Delete(data.Object, string) error {
	return derp.Internal("test", "unused")
}

// HardDelete implements the data.Collection interface.
func (c *outboxExportCollection) HardDelete(exp.Expression) error {
	return derp.Internal("test", "unused")
}

// outboxExportSession hands out the in-memory export collection.
type outboxExportSession struct {
	collection *outboxExportCollection
}

// Collection implements the data.Session interface.
func (s outboxExportSession) Collection(string) data.Collection { return s.collection }

// Context implements the data.Session interface.
func (s outboxExportSession) Context() context.Context { return context.Background() }

// Close implements the data.Session interface.
func (s outboxExportSession) Close() {}

// TestOutbox_ExportCollectionUsesActorID verifies that export collection queries use the
// persisted ownership field and do not return another actor's messages.
func TestOutbox_ExportCollectionUsesActorID(t *testing.T) {
	ownerID := primitive.NewObjectID()
	otherOwnerID := primitive.NewObjectID()
	ownedMessage := model.NewOutboxMessage()
	ownedMessage.ActorID = ownerID
	otherMessage := model.NewOutboxMessage()
	otherMessage.ActorID = otherOwnerID

	service := Outbox{}
	session := outboxExportSession{collection: &outboxExportCollection{records: []model.OutboxMessage{
		ownedMessage,
		otherMessage,
	}}}

	result, err := service.ExportCollection(session, ownerID)

	require.NoError(t, err)
	require.Equal(t, []model.IDOnly{{ID: ownedMessage.OutboxMessageID}}, result)
}

// TestOutbox_ExportCollectionReturnsEmptyForUnknownActor verifies that an actor with no
// outbox records receives an empty result without exposing another actor's records.
func TestOutbox_ExportCollectionReturnsEmptyForUnknownActor(t *testing.T) {
	ownerID := primitive.NewObjectID()
	message := model.NewOutboxMessage()
	message.ActorID = primitive.NewObjectID()

	service := Outbox{}
	session := outboxExportSession{collection: &outboxExportCollection{records: []model.OutboxMessage{message}}}

	result, err := service.ExportCollection(session, ownerID)

	require.NoError(t, err)
	require.Empty(t, result)
}

// TestOutbox_getActorRejectsUnroutableTypes pins the set of Actors that may own an Outbox message.
// Application, Search, and SearchDomain have no route that serves an Outbox item, so a message they
// own could never be dereferenced by any peer (BUG-146).
func TestOutbox_getActorRejectsUnroutableTypes(t *testing.T) {

	service := Outbox{}

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	unroutable := []string{
		model.FollowerTypeApplication,
		model.FollowerTypeSearch,
		model.FollowerTypeSearchDomain,
		"",
		"NOT-A-REAL-ACTOR-TYPE",
	}

	for _, actorType := range unroutable {
		_, err := service.getActor(session, actorType, primitive.NewObjectID())
		require.Error(t, err, "actorType %q must not be allowed to own an Outbox message", actorType)
	}
}

// TestOutbox_SaveRequiresAnActor verifies that a message whose Actor cannot be resolved is never
// written. Before BUG-146 such a message stored an empty ActorURL, then published an empty `actor`
// and the relative id "/pub/outbox/<id>" that no peer can dereference.
func TestOutbox_SaveRequiresAnActor(t *testing.T) {

	service := Outbox{}

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	message := model.NewOutboxMessage()
	message.ActorType = model.FollowerTypeApplication
	message.ActorID = primitive.NewObjectID()

	require.Error(t, service.Save(session, &message, "Testing"))

	// The message is left unidentifiable, rather than carrying a relative ID into the database
	require.Empty(t, message.ActorURL)
	require.Empty(t, message.ActivityURL)
	require.Empty(t, message.ActivityPubURL())
}

// TestOutbox_calcActivityURL verifies that an activity URL is built from the Actor's own URL, and
// that a message with no Actor mints nothing at all
func TestOutbox_calcActivityURL(t *testing.T) {

	service := Outbox{}

	messageID, err := primitive.ObjectIDFromHex("687aeeef6c699789c9ce623a")
	require.NoError(t, err)

	testCases := []struct {
		name     string
		actorURL string
		expected string
	}{
		{"user actor", "https://x.social/@bob", "https://x.social/@bob/pub/outbox/687aeeef6c699789c9ce623a"},
		{"stream actor", "https://x.social/123", "https://x.social/123/pub/outbox/687aeeef6c699789c9ce623a"},
		{"no actor", "", ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {

			message := model.NewOutboxMessage()
			message.OutboxMessageID = messageID
			message.ActorURL = testCase.actorURL

			require.Equal(t, testCase.expected, service.calcActivityURL(&message))
		})
	}
}
