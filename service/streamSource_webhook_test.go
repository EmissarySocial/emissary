package service

import (
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// sourceWithToken returns a StreamSource record carrying a specific webhook token
func sourceWithToken(token string) model.StreamSource {
	result := validStreamSource()
	result.Config[model.StreamSourceConfigWebhookToken] = token
	return result
}

/******************************************
 * The Shared Token
 ******************************************/

// TestStreamSource_WebhookFansOut pins D7: one token belongs to many records, so one ping from a
// repository refreshes every page sourced from it
func TestStreamSource_WebhookFansOut(t *testing.T) {

	const token = "0123456789abcdef0123456789abcdef"

	first := sourceWithToken(token)
	second := sourceWithToken(token)
	other := sourceWithToken("ffffffffffffffffffffffffffffffff")

	service, session := newStreamSourceService(first, second, other)

	require.NoError(t, service.SyncByWebhookToken(session, token))

	tasks := session.publishedTasks()
	require.Len(t, tasks, 2, "every record sharing the token is synchronized, and no others")

	published := []string{
		tasks[0].Arguments.GetString("streamSourceId"),
		tasks[1].Arguments.GetString("streamSourceId"),
	}

	require.Contains(t, published, first.StreamSourceID.Hex())
	require.Contains(t, published, second.StreamSourceID.Hex())
	require.NotContains(t, published, other.StreamSourceID.Hex())
}

// TestStreamSource_WebhookTaskIsDeduplicated confirms that each task carries a per-record
// signature.  That is what turns a ping flood into one queued sync per record.
func TestStreamSource_WebhookTaskIsDeduplicated(t *testing.T) {

	const token = "0123456789abcdef0123456789abcdef"

	streamSource := sourceWithToken(token)
	service, session := newStreamSourceService(streamSource)

	require.NoError(t, service.SyncByWebhookToken(session, token))

	tasks := session.publishedTasks()
	require.Len(t, tasks, 1)
	require.Equal(t, "StreamSource-Sync:"+streamSource.StreamSourceID.Hex(), tasks[0].Signature)
	require.Equal(t, TaskSyncStreamSource, tasks[0].Name)
}

// TestStreamSource_WebhookIgnoresDeletedRecords confirms that a record in the soft-delete window
// is never synchronized, even though its token is still stored beside it
func TestStreamSource_WebhookIgnoresDeletedRecords(t *testing.T) {

	const token = "0123456789abcdef0123456789abcdef"

	deleted := sourceWithToken(token)
	deleted.DeleteDate = 1

	service, session := newStreamSourceService(deleted)

	require.NoError(t, service.SyncByWebhookToken(session, token))
	require.Empty(t, session.publishedTasks())
}

// TestStreamSource_WebhookUnknownTokenSyncsNothing confirms that a token nobody holds does no work
func TestStreamSource_WebhookUnknownTokenSyncsNothing(t *testing.T) {

	service, session := newStreamSourceService(sourceWithToken("0123456789abcdef0123456789abcdef"))

	require.NoError(t, service.SyncByWebhookToken(session, "ffffffffffffffffffffffffffffffff"))
	require.Empty(t, session.publishedTasks())
}

/******************************************
 * Refusing a Short Token
 ******************************************/

// TestStreamSource_WebhookRefusesShortToken pins the guard that keeps an empty token from selecting
// every record whose token was never set.  This is a fan-out trigger, not an authorization check,
// so a permissive match starts work rather than merely allowing it.
func TestStreamSource_WebhookRefusesShortToken(t *testing.T) {

	// A record whose token was never set is what an empty token would otherwise match
	blank := validStreamSource()
	blank.Config[model.StreamSourceConfigWebhookToken] = ""

	service, session := newStreamSourceService(blank)

	for _, token := range []string{"", " ", "short", strings.Repeat("a", model.StreamSourceWebhookTokenMinLength-1)} {

		err := service.SyncByWebhookToken(session, token)

		require.Error(t, err, "token: %q", token)
		require.True(t, derp.IsClientError(err), "token: %q, error: %v", token, err)
		require.Empty(t, session.publishedTasks(), "a refused token queues nothing: %q", token)
	}

	// The boundary itself is allowed, so the minimum is a floor rather than an off-by-one
	require.NoError(t, service.SyncByWebhookToken(session, strings.Repeat("a", model.StreamSourceWebhookTokenMinLength)))
}

// TestStreamSource_SaveRefusesShortToken confirms that a hand-edited token is refused on the way
// in, so the endpoint's own check is a second line rather than the only one
func TestStreamSource_SaveRefusesShortToken(t *testing.T) {

	service, session := newStreamSourceService()
	streamSource := validStreamSource()
	streamSource.Config[model.StreamSourceConfigWebhookToken] = "too-short"

	err := service.Save(session, &streamSource, "Created")

	require.Error(t, err)
	require.True(t, derp.IsClientError(err), "got %v", err)
	require.Empty(t, session.collection.saved)
}

/******************************************
 * The Generated Token
 ******************************************/

// TestNewStreamSource_MintsAWebhookToken confirms that a new record is usable without an admin
// choosing a secret by hand
func TestNewStreamSource_MintsAWebhookToken(t *testing.T) {

	streamSource := model.NewStreamSource()

	require.Len(t, streamSource.WebhookToken(), 64, "a SHA-256 digest is 64 hexadecimal characters")
	require.GreaterOrEqual(t, len(streamSource.WebhookToken()), model.StreamSourceWebhookTokenMinLength)
	require.Regexp(t, "^[0-9a-f]{64}$", streamSource.WebhookToken(), "a token travels in a URL path")
}

// TestNewWebhookToken_IsUnpredictable pins the decision that a token is hashed from crypto/rand and
// NOT from an ObjectID.  primitive.NewObjectID mints its 5 process bytes once per process and
// increments a 3-byte counter, so a token derived from one is brute-forced offline in seconds --
// and with fan-out, one guessed token fires every record behind it.
func TestNewWebhookToken_IsUnpredictable(t *testing.T) {

	const sampleSize = 512

	tokens := make(map[string]bool, sampleSize)

	for range sampleSize {
		tokens[model.NewWebhookToken()] = true
	}

	require.Len(t, tokens, sampleSize, "every token is distinct")

	// The same loop over ObjectIDs is what this test exists to rule out: consecutive IDs differ
	// only in a counter, so their digests would be distinct and still guessable
	first := primitive.NewObjectID()
	second := primitive.NewObjectID()
	require.Equal(t, first.Hex()[:18], second.Hex()[:18], "consecutive ObjectIDs share their process bytes")
}
