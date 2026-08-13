package ascache

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// countingClient is a streams.Client that stands in for the origin server, counting how many times
// it was actually contacted.
type countingClient struct {
	calls        int    // Number of times Load reached this client
	publicKeyPEM string // Value returned as the Actor's key, so a "rotation" can be simulated
}

// SetRootClient satisfies streams.Client.  This client makes no recursive calls, so it needs no root.
func (client *countingClient) SetRootClient(streams.Client) {}

// Load records the call and returns a minimal, cacheable Actor document.
func (client *countingClient) Load(uri string, options ...any) (streams.Document, error) {

	client.calls++

	header := make(http.Header)
	header.Set(cacheheader.HeaderCacheControl, "max-age=3600")

	document := streams.NewDocument(
		mapof.Any{
			"id":           uri,
			"type":         "Person",
			"publicKeyPem": client.publicKeyPEM,
		},
		streams.WithHTTPHeader(header),
	)

	return document, nil
}

// Save satisfies streams.Client.  The origin does not store anything.
func (client *countingClient) Save(streams.Document) error { return nil }

// Delete satisfies streams.Client.  The origin does not store anything.
func (client *countingClient) Delete(string) error { return nil }

// newTestClient returns a cache Client backed by an in-memory database, plus the origin it fronts.
func newTestClient() (*Client, *countingClient) {

	inner := &countingClient{publicKeyPEM: "PEM-ORIGINAL"}
	client := New(inner, nil, newFakeServer(), "Application", primitive.NilObjectID, "test.example")

	return client, inner
}

// ageCachedValue rewinds a cached entry's "Received" stamp, simulating the passage of time.
func ageCachedValue(t *testing.T, client *Client, url string, age time.Duration) {

	t.Helper()

	session, err := client.commonDatabase.Session(context.Background())
	require.NoError(t, err)

	value := NewValue()
	require.NoError(t, client.loadByURL(session, url, &value))

	value.Received = time.Now().Add(-age).Unix()
	require.NoError(t, client.collection(session).Save(&value, "aging for test"))
}

// TestClient_Load_BurstCollapse is the amplification fix stated directly: repeated reads of the same
// document contact the origin exactly once.
func TestClient_Load_BurstCollapse(t *testing.T) {

	client, origin := newTestClient()

	for range 25 {
		_, err := client.Load("https://remote.example/@alice")
		require.NoError(t, err)
	}

	require.Equal(t, 1, origin.calls)
}

// TestClient_Load_WriteOnlyAlwaysFetches pins the behavior that made BUG-22 an amplifier: WithWriteOnly
// skips the cache read, so every call is an outbound request.
func TestClient_Load_WriteOnlyAlwaysFetches(t *testing.T) {

	// This is why the public-key finder no longer uses WithWriteOnly on its own. The mode is still
	// correct for a caller that genuinely wants a forced refresh -- it just must be bounded.
	client, origin := newTestClient()

	for range 5 {
		_, err := client.Load("https://remote.example/@alice", WithWriteOnly())
		require.NoError(t, err)
	}

	require.Equal(t, 5, origin.calls)
}

// TestClient_Load_MinAgeCapsForcedReload is the assertion that the cooldown works: a repeated FORCED
// reload inside the window produces no outbound traffic at all.
func TestClient_Load_MinAgeCapsForcedReload(t *testing.T) {

	// Without this, a repeated bad signature buys the sender one fetch per request, aimed at whatever
	// host their keyId names. (BUG-22 D3)
	client, origin := newTestClient()

	_, err := client.Load("https://remote.example/@alice")
	require.NoError(t, err)
	require.Equal(t, 1, origin.calls)

	for range 100 {
		_, err := client.Load("https://remote.example/@alice", WithWriteOnly(), WithMinAge(time.Minute))
		require.NoError(t, err)
	}

	require.Equal(t, 1, origin.calls)
}

// TestClient_Load_MinAgeExpiredAllowsReload confirms the cooldown is a delay and not a block: once the
// entry is older than minAge, a forced reload reaches the origin again.
func TestClient_Load_MinAgeExpiredAllowsReload(t *testing.T) {

	client, origin := newTestClient()

	_, err := client.Load("https://remote.example/@alice")
	require.NoError(t, err)

	ageCachedValue(t, client, "https://remote.example/@alice", 5*time.Minute)

	_, err = client.Load("https://remote.example/@alice", WithWriteOnly(), WithMinAge(time.Minute))
	require.NoError(t, err)

	require.Equal(t, 2, origin.calls)
}

// TestClient_Load_MinAgeSeesRotatedKey is the rotation property, end to end: a key that changes at the
// origin is picked up by a forced reload once the cooldown has passed.
func TestClient_Load_MinAgeSeesRotatedKey(t *testing.T) {

	client, origin := newTestClient()

	first, err := client.Load("https://remote.example/@alice")
	require.NoError(t, err)
	require.Equal(t, "PEM-ORIGINAL", first.PublicKeyPEM())

	// The remote rotates its key
	origin.publicKeyPEM = "PEM-ROTATED"

	// Inside the cooldown, the reload is suppressed and the OLD key is still served
	suppressed, err := client.Load("https://remote.example/@alice", WithWriteOnly(), WithMinAge(time.Minute))
	require.NoError(t, err)
	require.Equal(t, "PEM-ORIGINAL", suppressed.PublicKeyPEM())

	// Past the cooldown, the reload happens and the NEW key arrives
	ageCachedValue(t, client, "https://remote.example/@alice", 5*time.Minute)

	rotated, err := client.Load("https://remote.example/@alice", WithWriteOnly(), WithMinAge(time.Minute))
	require.NoError(t, err)
	require.Equal(t, "PEM-ROTATED", rotated.PublicKeyPEM())
}

// TestClient_Load_CacheHitIsStamped pins the signal that the signature verifier reads: an answer served
// from the cache is marked as such, on both the normal read path and the cooldown short-circuit.
func TestClient_Load_CacheHitIsStamped(t *testing.T) {

	// service.ActivityStream.VerifySignature refuses to retry a verification unless the key it used
	// came from the cache, so losing this stamp would silently disable the rotation repair.
	client, _ := newTestClient()

	fromOrigin, err := client.Load("https://remote.example/@alice")
	require.NoError(t, err)
	require.False(t, FromCache(fromOrigin))

	fromCache, err := client.Load("https://remote.example/@alice")
	require.NoError(t, err)
	require.True(t, FromCache(fromCache))

	fromCooldown, err := client.Load("https://remote.example/@alice", WithWriteOnly(), WithMinAge(time.Minute))
	require.NoError(t, err)
	require.True(t, FromCache(fromCooldown))
}
