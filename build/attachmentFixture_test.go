package build

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/exp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const attachmentTestConnection = "mongodb://localhost:27017/?directConnection=true"

// newAttachmentTestSession returns a session on a throwaway database in the local replica set, and
// skips the test when there is none. The Attachment service filters every read on the journal's
// deleteDate, which data-mock cannot match, so these tests need a real database.
func newAttachmentTestSession(t *testing.T) data.Session {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(attachmentTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + attachmentTestConnection)
	}

	// RULE: Transactions need a replica set, and a standalone server names none
	hello := bson.M{}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil || hello["setName"] == nil {
		_ = client.Disconnect(ctx) // Nothing to report: the test is skipped either way
		t.Skip("Skipping: no MongoDB replica set at " + attachmentTestConnection)
	}

	database := client.Database("emissary_attachmenttest_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = database.Drop(ctx)     // A leftover test database is harmless
		_ = client.Disconnect(ctx) // ...and so is a connection closed by process exit
	})

	sessionCtx, cancelSession := context.WithCancel(context.Background())
	t.Cleanup(cancelSession)

	session, err := mongodb.NewServer(database).Session(sessionCtx)
	require.NoError(t, err)

	return session
}

// stubAttachmentFactory satisfies the Factory interface by embedding it (nil) and overriding only Attachment()
type stubAttachmentFactory struct {
	Factory
	attachment *service.Attachment
}

// Attachment returns the Attachment service under test. Implements the Factory interface.
func (factory stubAttachmentFactory) Attachment() *service.Attachment {
	return factory.attachment
}

// stubStreamService satisfies the ModelService interface by embedding it (nil) and overriding only ObjectType()
type stubStreamService struct {
	service.ModelService
}

// ObjectType names the kind of object being built. Implements the ModelService interface.
func (stubStreamService) ObjectType() string {
	return "Stream"
}

// stubAttachmentBuilder satisfies the Builder interface by embedding it (nil) and overriding the
// five methods the attachment steps reach for
type stubAttachmentBuilder struct {
	Builder
	_request  *http.Request
	_session  data.Session
	_factory  Factory
	_objectID primitive.ObjectID
}

// request returns the POST being handled. Implements the Builder interface.
func (builder stubAttachmentBuilder) request() *http.Request {
	return builder._request
}

// session returns the test database session. Implements the Builder interface.
func (builder stubAttachmentBuilder) session() data.Session {
	return builder._session
}

// factory returns the stubbed factory. Implements the Builder interface.
func (builder stubAttachmentBuilder) factory() Factory {
	return builder._factory
}

// objectID returns the ID of the Stream whose attachments are under test. Implements the Builder interface.
func (builder stubAttachmentBuilder) objectID() primitive.ObjectID {
	return builder._objectID
}

// service returns a model service that names the Stream type. Implements the Builder interface.
func (builder stubAttachmentBuilder) service() service.ModelService {
	return stubStreamService{}
}

// attachmentFixture holds "photo" Attachments on one Stream in a throwaway database,
// keyed by the rank each started with
type attachmentFixture struct {
	session  data.Session
	service  *service.Attachment
	objectID primitive.ObjectID
	byRank   map[int]primitive.ObjectID
}

// newAttachmentFixture stores one "photo" Attachment for each rank given
func newAttachmentFixture(t *testing.T, ranks ...int) attachmentFixture {
	t.Helper()

	attachmentService := service.NewAttachment()

	fixture := attachmentFixture{
		session:  newAttachmentTestSession(t),
		service:  &attachmentService,
		objectID: primitive.NewObjectID(),
		byRank:   map[int]primitive.ObjectID{},
	}

	for _, rank := range ranks {
		fixture.byRank[rank] = fixture.store(t, "photo", rank, "")
	}

	return fixture
}

// store writes one Attachment on the fixture's Stream and returns its ID. A status outside
// the schema's enum makes the next validated save of it fail.
func (fixture attachmentFixture) store(t *testing.T, category string, rank int, status string) primitive.ObjectID {
	t.Helper()

	attachment := model.NewAttachment("Stream", fixture.objectID)
	attachment.Category = category
	attachment.Original = "file.jpg"
	attachment.Rank = rank
	attachment.Status = status
	require.NoError(t, fixture.session.Collection("Attachment").Save(&attachment, "seed"))

	return attachment.AttachmentID
}

// hex returns the hex ID of the Attachment that started at the given rank
func (fixture attachmentFixture) hex(originalRank int) string {
	return fixture.byRank[originalRank].Hex()
}

// rankOf reads the stored rank of the Attachment that started at the given rank
func (fixture attachmentFixture) rankOf(t *testing.T, originalRank int) int {
	t.Helper()
	return fixture.rankByID(t, fixture.byRank[originalRank])
}

// rankByID reads the stored rank of one Attachment
func (fixture attachmentFixture) rankByID(t *testing.T, attachmentID primitive.ObjectID) int {
	t.Helper()

	attachment := model.NewEmptyAttachment()
	require.NoError(t, fixture.session.Collection("Attachment").Load(exp.Equal("_id", attachmentID), &attachment))

	return attachment.Rank
}
