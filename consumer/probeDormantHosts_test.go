package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/benpate/floodgate"
	"github.com/benpate/turbine/queue"
	"github.com/stretchr/testify/require"
)

// scriptedProber answers each host from a fixed table, and records which hosts it was asked about
type scriptedProber struct {
	alive  map[string]string
	probed []string
}

// probe returns the software of a host that is alive, and an error for any other
func (prober *scriptedProber) probe(host string) (string, string, error) {

	prober.probed = append(prober.probed, host)

	software, found := prober.alive[host]

	if !found {
		return "", "", errors.New("connection refused")
	}

	return software, "1.0", nil
}

// newDormantGate returns a Gate whose named hosts are dormant and due for a check
func newDormantGate(t *testing.T, hosts ...string) (floodgate.Gate, *floodgate.MemoryStore, *time.Time) {

	t.Helper()

	current := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	store := floodgate.NewMemoryStore()
	gate := floodgate.New(store, floodgate.WithDormantDays(1), floodgate.WithClock(func() time.Time { return current }))

	for _, host := range hosts {
		require.NoError(t, gate.Failure(context.Background(), host))
	}

	// The first check is due a week after the host was marked
	current = current.Add(7 * 24 * time.Hour)

	return gate, store, &current
}

// TestProbeDormantHosts verifies that a host that answers is revived, and that one that does
// not is checked again later
func TestProbeDormantHosts(t *testing.T) {

	gate, store, current := newDormantGate(t, "back.example", "gone.example")
	prober := &scriptedProber{alive: map[string]string{"back.example": "mastodon"}}
	ctx := context.Background()

	result := probeDormantHosts(ctx, gate, prober)
	require.Equal(t, queue.ResultStatusSuccess, result.Status)
	require.ElementsMatch(t, []string{"back.example", "gone.example"}, prober.probed)

	back, err := store.Load(ctx, "back.example")
	require.NoError(t, err)
	require.False(t, back.IsDormant())
	require.Equal(t, "mastodon", back.Software)
	require.Equal(t, "1.0", back.Version)

	gone, err := store.Load(ctx, "gone.example")
	require.NoError(t, err)
	require.True(t, gone.IsDormant())
	require.Equal(t, 1, gone.ProbeCount)
	require.Equal(t, current.Add(7*24*time.Hour).Unix(), gone.NextProbe)

	// Neither host is due again today
	prober.probed = nil
	require.Equal(t, queue.ResultStatusSuccess, probeDormantHosts(ctx, gate, prober).Status)
	require.Empty(t, prober.probed)
}

// TestProbeDormantHosts_StopsWhenTimeIsUp verifies that an expired task checks no more hosts
func TestProbeDormantHosts_StopsWhenTimeIsUp(t *testing.T) {

	gate, _, _ := newDormantGate(t, "one.example", "two.example")
	prober := &scriptedProber{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Equal(t, queue.ResultStatusSuccess, probeDormantHosts(ctx, gate, prober).Status)
	require.Empty(t, prober.probed)
}

// TestProbeDormantHosts_IsRegistered confirms that the task reaches a handler and runs promptly enough
func TestProbeDormantHosts_IsRegistered(t *testing.T) {

	task := queue.NewTask(TaskProbeDormantHosts, nil)

	result := New(unresolvableFactory{}).Run(task)
	require.Equal(t, queue.ResultStatusSuccess, result.Status, "no host is due, so there is nothing to do")

	task.Priority = -1
	require.NoError(t, PreProcessor(&task))

	// RULE: Assert the EXACT value.  A name that matches no case leaves Priority at the -1 sentinel.
	require.Equal(t, 1024, task.Priority)
}

// TestNodeInfoDocumentURL verifies that only an https link on the host itself is followed
func TestNodeInfoDocumentURL(t *testing.T) {

	discovery := func(links ...string) nodeInfoDiscovery {
		result := nodeInfoDiscovery{}
		for _, href := range links {
			result.Links = append(result.Links, struct {
				Href string `json:"href"`
			}{Href: href})
		}
		return result
	}

	require.Empty(t, nodeInfoDocumentURL("remote.example", discovery()))
	require.Empty(t, nodeInfoDocumentURL("remote.example", discovery("https://other.example/nodeinfo/2.0")))
	require.Empty(t, nodeInfoDocumentURL("remote.example", discovery("http://remote.example/nodeinfo/2.0")))
	require.Empty(t, nodeInfoDocumentURL("remote.example", discovery("://bad")))

	// The last matching link wins, which is the newest schema that the host lists
	require.Equal(t,
		"https://REMOTE.example/nodeinfo/2.1",
		nodeInfoDocumentURL("remote.example", discovery("https://remote.example/nodeinfo/2.0", "https://169.254.169.254/latest", "https://REMOTE.example/nodeinfo/2.1")),
	)
}
