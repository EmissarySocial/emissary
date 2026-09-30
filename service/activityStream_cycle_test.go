package service

/******************************************
 * Client Stack Cycle Tests
 *
 * A remote document can lead a load back to a URL that the
 * same stack is already loading.  These tests serve each such
 * shape from a real HTTP server and load it through the real
 * client layers, so a hang or a runaway chain fails the test.
 ******************************************/

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/ascacherules"
	"github.com/EmissarySocial/emissary/tools/ashash"
	"github.com/EmissarySocial/emissary/tools/asnormalizer"
	"github.com/EmissarySocial/emissary/tools/assanitizer"
	"github.com/benpate/hannibal/clients"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/remote"
	"github.com/benpate/sherlock/activitypub"
	"github.com/benpate/sherlock/bridgyfed"
	"github.com/benpate/sherlock/tagspub"
	"github.com/benpate/sherlock/tombstone"
	"github.com/benpate/sherlock/webfinger"
	"github.com/stretchr/testify/require"
)

// cycleTimeout is how long one load may run before the test calls it a hang
const cycleTimeout = 5 * time.Second

// cycleChainLength is how many links the fresh-URL chain server will serve before answering 404
const cycleChainLength = 50

// webfingerPath is where the test server answers the WebFinger lookups that Bridgy Fed would receive
const webfingerPath = "/.well-known/webfinger"

// cycleSigner is the identity every test stack signs as, so that stacks sharing a Carpool merge
const cycleSigner = "cycle-test"

// cycleCallers is how many stacks load at once in the concurrent tests
const cycleCallers = 10

// cycleDelay is how long the server holds each response in the concurrent tests, so that every
// caller arrives while the first load is still running
const cycleDelay = 200 * time.Millisecond

// TestClientCycle_SelfAttributedNote confirms that a Note whose author is the Note itself loads,
// and is fetched once.
func TestClientCycle_SelfAttributedNote(t *testing.T) {

	// BUG-212: the author lookup re-entered the Carpool as a rider on its own load.
	server := newCycleServer(t)
	noteURL := server.url("/notes/self")

	server.serve("/notes/self", map[string]any{
		"id":           noteURL,
		"type":         "Note",
		"attributedTo": noteURL,
		"content":      "I wrote myself",
	})

	document := loadWithin(t, cycleStack(server), noteURL)

	require.Equal(t, noteURL, document.ID())
	require.Equal(t, 1, server.requests("/notes/self"))
}

// TestClientCycle_MutuallyAttributedNotes confirms that two Notes naming each other as author both
// load, and each is fetched once.
func TestClientCycle_MutuallyAttributedNotes(t *testing.T) {

	// BUG-212: the cycle closes one level deeper than a self-reference.
	server := newCycleServer(t)
	firstURL := server.url("/notes/a")
	secondURL := server.url("/notes/b")

	server.serve("/notes/a", map[string]any{
		"id":           firstURL,
		"type":         "Note",
		"attributedTo": secondURL,
		"content":      "B wrote me",
	})

	server.serve("/notes/b", map[string]any{
		"id":           secondURL,
		"type":         "Note",
		"attributedTo": firstURL,
		"content":      "A wrote me",
	})

	document := loadWithin(t, cycleStack(server), firstURL)

	require.Equal(t, firstURL, document.ID())
	require.Equal(t, 1, server.requests("/notes/a"))
	require.Equal(t, 1, server.requests("/notes/b"))
}

// TestClientCycle_CreateOfItself confirms that a Create whose object is its own id, given as a bare
// URL, loads, and is fetched once.
func TestClientCycle_CreateOfItself(t *testing.T) {

	// BUG-212: no explicit load is involved. Unwrapping the Create reads the object's type,
	// and a getter on a bare-URL value loads that URL through the top of the stack.
	server := newCycleServer(t)
	createURL := server.url("/activities/self")

	server.serve("/activities/self", map[string]any{
		"id":     createURL,
		"type":   "Create",
		"actor":  server.url("/actors/alice"),
		"object": createURL,
	})

	server.serve("/actors/alice", cycleActor(server.url("/actors/alice")))

	document := loadWithin(t, cycleStack(server), createURL)

	require.Equal(t, createURL, document.ID())
	require.Equal(t, 1, server.requests("/activities/self"))
}

// TestClientCycle_ActorWithBareKeyLink confirms that an actor whose publicKey is a bare link to its
// own key fragment loads, and is fetched once.
func TestClientCycle_ActorWithBareKeyLink(t *testing.T) {

	// BUG-212: the one shape here that a well-behaved server may publish. Reading the key
	// loads "#main-key", ashash strips the fragment, and the load lands on the actor itself.
	server := newCycleServer(t)
	actorURL := server.url("/actors/keyself")

	actor := cycleActor(actorURL)
	actor["publicKey"] = actorURL + "#main-key"
	server.serve("/actors/keyself", actor)

	document := loadWithin(t, cycleStack(server), actorURL)

	require.Equal(t, actorURL, document.ID())
	require.Equal(t, 1, server.requests("/actors/keyself"))
}

// TestClientCycle_FreshURLChain confirms that a chain of Notes, each attributed to a new URL, is
// followed to its end with every link fetched once.
func TestClientCycle_FreshURLChain(t *testing.T) {

	// BUG-212: there is no depth limit, so only the server's end of the chain stops this.
	server := newCycleServer(t)
	server.serveChain(cycleChainLength)

	loadWithin(t, cycleStack(server), server.url("/chain/0"))

	// Every link, plus the missing one that ends the chain, is requested once
	for index := range cycleChainLength + 1 {
		require.Equal(t, 1, server.requests(fmt.Sprintf("/chain/%d", index)), "link %d", index)
	}
}

// TestClientCycle_BridgyFedStaysLocal confirms that the Bridgy Fed lookup the stack makes for every
// local URL reaches the test server's copy, not bsky.brid.gy.
func TestClientCycle_BridgyFedStaysLocal(t *testing.T) {

	server := newCycleServer(t)
	noteURL := server.url("/notes/plain")

	server.serve("/notes/plain", map[string]any{"id": noteURL, "type": "Note", "content": "Hello"})

	document := loadWithin(t, cycleStack(server), noteURL)

	require.Equal(t, noteURL, document.ID())
	require.Positive(t, server.requests(webfingerPath))
}

// TestClientCycle_ConcurrentMutualAuthors confirms that two stacks sharing one Carpool, each loading
// one of two Notes that name each other as author, both return.
func TestClientCycle_ConcurrentMutualAuthors(t *testing.T) {

	// BUG-212: each leader's author lookup would wait as a rider on the other's load, forever.
	server := newCycleServer(t)
	server.setDelay(cycleDelay)
	firstURL := server.url("/notes/a")
	secondURL := server.url("/notes/b")

	server.serve("/notes/a", map[string]any{"id": firstURL, "type": "Note", "attributedTo": secondURL})
	server.serve("/notes/b", map[string]any{"id": secondURL, "type": "Note", "attributedTo": firstURL})

	carpool := clients.NewCarpool()
	newStack := func() streams.Client { return cycleStackOn(server, carpool) }

	documents := loadConcurrently(t, newStack, firstURL, secondURL)

	require.Equal(t, firstURL, documents[0].ID())
	require.Equal(t, secondURL, documents[1].ID())
}

// TestClientCycle_ConcurrentSelfAttributedNote confirms that many stacks sharing one Carpool, all
// loading a self-attributed Note at once, all return from a single fetch.
func TestClientCycle_ConcurrentSelfAttributedNote(t *testing.T) {

	// BUG-212: the leader never finished, so every rider behind it waited too.
	server := newCycleServer(t)
	server.setDelay(cycleDelay)
	noteURL := server.url("/notes/self")

	server.serve("/notes/self", map[string]any{"id": noteURL, "type": "Note", "attributedTo": noteURL})

	carpool := clients.NewCarpool()
	newStack := func() streams.Client { return cycleStackOn(server, carpool) }

	for _, document := range loadConcurrently(t, newStack, slices.Repeat([]string{noteURL}, cycleCallers)...) {
		require.Equal(t, noteURL, document.ID())
	}

	require.Equal(t, 1, server.requests("/notes/self"))
}

// TestClientCycle_TopLevelLoadsStillMerge confirms that concurrent loads of an ordinary Note share one
// fetch of the Note and one fetch of its author, which is the Carpool's purpose.
func TestClientCycle_TopLevelLoadsStillMerge(t *testing.T) {

	// BUG-181: nested loads skip the Carpool, but top-level loads must still merge.
	server := newCycleServer(t)
	server.setDelay(cycleDelay)
	noteURL := server.url("/notes/plain")
	authorURL := server.url("/actors/alice")

	server.serve("/notes/plain", map[string]any{"id": noteURL, "type": "Note", "attributedTo": authorURL})
	server.serve("/actors/alice", cycleActor(authorURL))

	carpool := clients.NewCarpool()
	newStack := func() streams.Client { return cycleStackOn(server, carpool) }

	for _, document := range loadConcurrently(t, newStack, slices.Repeat([]string{noteURL}, cycleCallers)...) {
		require.Equal(t, noteURL, document.ID())
		require.Equal(t, "Cycle Tester", document.AttributedTo().Name())
	}

	require.Equal(t, 1, server.requests("/notes/plain"))
	require.Equal(t, 1, server.requests("/actors/alice"))
}

/******************************************
 * Helpers
 ******************************************/

// cycleStack builds the client layers of service.ActivityStream.Client, in the same order, around a
// fresh Carpool.  ascache and asrules are omitted because both need a live database.
func cycleStack(server *cycleServer) streams.Client {
	return cycleStackOn(server, clients.NewCarpool())
}

// cycleStackOn builds the same stack as cycleStack around a Carpool that other stacks may share, as
// every stack in one server process does.
func cycleStackOn(server *cycleServer, carpool *clients.Carpool) streams.Client {

	// RULE: This MUST mirror service.ActivityStream.Client, because a cycle is made by the layers
	// together.  Omitting ascache is the same as a cache miss, which is where every cycle begins.
	client := carpool.Client(cycleLowerLayers(server), cycleSigner)
	return ashash.New(client)
}

// cycleLowerLayers builds the layers of service.ActivityStream.Client below ascache, in the same
// order, with Bridgy Fed's lookups sent to the test server
func cycleLowerLayers(server *cycleServer) streams.Client {

	// httptest serves from 127.0.0.1, which the transport's SSRF guard refuses by default
	const allowPrivateIPs = true

	client := activitypub.New(
		activitypub.WithUserAgent("emissary-test"),
		activitypub.WithAllowPrivateIPs(allowPrivateIPs),
	)

	client = tombstone.New(client)

	client = webfinger.New(client, remote.Option{
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.AllowPrivateIPs(allowPrivateIPs)
			return nil
		},
	})

	// RULE: Bridgy Fed treats every 127.0.0.1 URL as a Bluesky handle and looks it up over the
	// internet, so the test server answers in its place, as bsky.brid.gy does
	client = bridgyfed.New(client, bridgyfed.WithHostname(server.host()))
	client = tagspub.New(client)
	client = assanitizer.New(client, model.NamespaceEmissary)
	client = asnormalizer.New(client)

	return ascacherules.New(client)
}

// loadWithin loads a URL through the client, failing the test if the load has not returned by
// cycleTimeout.  A load that never returns leaves its goroutine behind until the test binary exits.
func loadWithin(t *testing.T, client streams.Client, url string) streams.Document {

	t.Helper()

	type outcome struct {
		document streams.Document
		err      error
	}

	done := make(chan outcome, 1)

	go func() {
		document, err := client.Load(url)
		done <- outcome{document: document, err: err}
	}()

	select {

	case result := <-done:
		require.NoError(t, result.err, "loading %s", url)
		return result.document

	case <-time.After(cycleTimeout):
		require.FailNow(t, "load did not return", "loading %s hung for %s", url, cycleTimeout)
		return streams.NilDocument()
	}
}

// loadConcurrently loads every URL at once, each through its own stack from newStack, and fails the
// test unless all of them return by cycleTimeout.  Results are in the same order as the URLs.
func loadConcurrently(t *testing.T, newStack func() streams.Client, urls ...string) []streams.Document {

	t.Helper()

	type outcome struct {
		index    int
		document streams.Document
		err      error
	}

	done := make(chan outcome, len(urls))
	start := make(chan struct{})

	// Build every stack first, then release all of the loads together
	for index, url := range urls {
		client := newStack()

		go func() {
			<-start
			document, err := client.Load(url)
			done <- outcome{index: index, document: document, err: err}
		}()
	}

	close(start)

	result := make([]streams.Document, len(urls))
	timeout := time.After(cycleTimeout)

	for range urls {
		select {

		case finished := <-done:
			require.NoError(t, finished.err, "loading %s", urls[finished.index])
			result[finished.index] = finished.document

		case <-timeout:
			require.FailNow(t, "loads did not return", "concurrent loads hung for %s", cycleTimeout)
		}
	}

	return result
}

// cycleActor returns a minimal Person document at the given URL
func cycleActor(url string) map[string]any {
	return map[string]any{
		"id":                url,
		"type":              "Person",
		"name":              "Cycle Tester",
		"preferredUsername": "cycle",
		"inbox":             url + "/inbox",
		"outbox":            url + "/outbox",
	}
}

// cycleServer is an httptest server that answers with canned ActivityPub documents and counts every
// request it receives, by path.
type cycleServer struct {
	server    *httptest.Server
	mutex     sync.Mutex
	documents map[string]map[string]any
	counts    map[string]int
	chain     int
	delay     time.Duration
}

// newCycleServer starts a cycleServer that closes when the test ends
func newCycleServer(t *testing.T) *cycleServer {

	t.Helper()

	result := &cycleServer{
		documents: make(map[string]map[string]any),
		counts:    make(map[string]int),
	}

	result.server = httptest.NewServer(http.HandlerFunc(result.handle))
	t.Cleanup(result.server.Close)

	return result
}

// url returns the absolute URL of a path on this server
func (server *cycleServer) url(path string) string {
	return server.server.URL + path
}

// host returns this server's host and port
func (server *cycleServer) host() string {
	return strings.TrimPrefix(server.server.URL, "http://")
}

// serve registers a document to answer at a path
func (server *cycleServer) serve(path string, document map[string]any) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	document["@context"] = "https://www.w3.org/ns/activitystreams"
	server.documents[path] = document
}

// serveChain answers /chain/0 through /chain/{length-1} with Notes, each attributed to the next link
func (server *cycleServer) serveChain(length int) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.chain = length
}

// setDelay makes the server hold every response for the given time before answering
func (server *cycleServer) setDelay(delay time.Duration) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.delay = delay
}

// requests returns how many times a path has been requested
func (server *cycleServer) requests(path string) int {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	return server.counts[path]
}

// requestsWithPrefix returns how many requests were made for paths starting with prefix
func (server *cycleServer) requestsWithPrefix(prefix string) int {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	result := 0

	for path, count := range server.counts {
		if strings.HasPrefix(path, prefix) {
			result += count
		}
	}

	return result
}

// handle counts the request and writes the document registered for its path, or 404
func (server *cycleServer) handle(w http.ResponseWriter, r *http.Request) {

	server.mutex.Lock()
	server.counts[r.URL.Path]++
	document, found := server.documents[r.URL.Path]

	if !found {
		document, found = server.chainLink(r.URL.Path)
	}

	delay := server.delay
	server.mutex.Unlock()

	// Stand in for Bridgy Fed, which answers every lookup the stack makes here
	if r.URL.Path == webfingerPath {
		server.bridgyFedNotFound(w, r)
		return
	}

	// Hold the response outside the mutex, so that concurrent requests are all counted at once
	time.Sleep(delay)

	if !found {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/activity+json")

	// The client may hang up early on an error; a short write says nothing about the test
	_ = json.NewEncoder(w).Encode(document)
}

// bridgyFedNotFound answers a WebFinger lookup the way bsky.brid.gy answered one for a 127.0.0.1 URL
// on 2026-09-29: 404, as HTML, naming the account it could not find
func (server *cycleServer) bridgyFedNotFound(w http.ResponseWriter, r *http.Request) {

	// The resource is "acct:<url>@<bridgy host>", and Bridgy Fed names only the <url>
	account := strings.TrimPrefix(r.URL.Query().Get("resource"), "acct:")
	account = strings.TrimSuffix(account, "@"+server.host())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusNotFound)

	// The client may hang up early; a short write says nothing about the test
	_, _ = w.Write([]byte("No atproto user found for " + html.EscapeString(account)))
}

// chainLink builds the Note for one link of the fresh-URL chain.  The caller holds the mutex.
func (server *cycleServer) chainLink(path string) (map[string]any, bool) {

	var index int

	if _, err := fmt.Sscanf(path, "/chain/%d", &index); err != nil {
		return nil, false
	}

	if index < 0 || index >= server.chain {
		return nil, false
	}

	return map[string]any{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           server.url(path),
		"type":         "Note",
		"attributedTo": server.url(fmt.Sprintf("/chain/%d", index+1)),
		"content":      "link " + fmt.Sprint(index),
	}, true
}
