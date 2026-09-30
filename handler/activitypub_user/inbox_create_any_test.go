package activitypub_user

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// createTestClient is a streams.Client that serves fixed documents by URL
type createTestClient struct {
	documents map[string]mapof.Any
}

// SetRootClient implements the streams.Client interface
func (client *createTestClient) SetRootClient(streams.Client) {}

// Load implements the streams.Client interface, returning the fixed document for this URL
func (client *createTestClient) Load(url string, _ ...any) (streams.Document, error) {

	if document, exists := client.documents[url]; exists {
		return streams.NewDocument(document, streams.WithClient(client)), nil
	}

	return streams.NilDocument(), derp.NotFound("createTestClient.Load", "Document not found", url)
}

// Save implements the streams.Client interface
func (client *createTestClient) Save(streams.Document) error {
	return nil
}

// Delete implements the streams.Client interface
func (client *createTestClient) Delete(string) error {
	return nil
}

// publicCreate returns a public Create activity wrapping the provided object
func publicCreate(id string, object any) mapof.Any {

	result := mapof.Any{
		vocab.PropertyID:    id,
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyActor: "https://social.example.com/@sender",
		vocab.PropertyTo:    []any{vocab.NamespaceActivityStreamsPublic},
	}

	if object != nil {
		result[vocab.PropertyObject] = object
	}

	return result
}

// TestInboxCreateOrUpdate_DropsCreateWithoutObject confirms that a Create or Update that does not
// carry an object is accepted and dropped. The empty Context has no factory, so reaching any later
// step would panic.
func TestInboxCreateOrUpdate_DropsCreateWithoutObject(t *testing.T) {

	// BUG-225: every shape below must stop at the RULE, before the newsfeed and context steps.
	selfURL := "https://social.example.com/activities/self"
	loopA := "https://social.example.com/activities/a"
	loopB := "https://social.example.com/activities/b"

	client := &createTestClient{documents: map[string]mapof.Any{
		selfURL: publicCreate(selfURL, selfURL),
		loopA:   publicCreate(loopA, loopB),
		loopB:   publicCreate(loopB, loopA),
	}}

	shapes := map[string]mapof.Any{
		"no object":             publicCreate("https://social.example.com/activities/empty", nil),
		"embedded Create":       publicCreate("https://social.example.com/activities/outer", publicCreate("https://social.example.com/activities/inner", mapof.Any{vocab.PropertyType: vocab.ObjectTypeNote})),
		"Create of itself":      client.documents[selfURL],
		"two Creates in a loop": client.documents[loopA],
	}

	for name, value := range shapes {
		activity := streams.NewDocument(value, streams.WithClient(client))
		require.NoError(t, inbox_CreateOrUpdate(Context{}, activity), name)
	}
}

// TestInboxCreateOrUpdate_ReportsFailedLoad confirms that an object URL that cannot be loaded
// returns an error, so that the sender retries the delivery.
func TestInboxCreateOrUpdate_ReportsFailedLoad(t *testing.T) {

	client := &createTestClient{documents: map[string]mapof.Any{}}
	value := publicCreate("https://social.example.com/activities/missing", "https://social.example.com/notes/missing")

	err := inbox_CreateOrUpdate(Context{}, streams.NewDocument(value, streams.WithClient(client)))
	require.True(t, derp.IsNotFound(err))
}
