package asnormalizer

import (
	"strconv"
	"sync"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// runawayLimit is how many times fakeInner serves one URL before refusing, so that a broken cycle
// check fails a test instead of overflowing the stack.  It is high because the fake has no cache, so
// every getter on a bare-URL value fetches that URL again.
const runawayLimit = 100

// TestClient_AuthorLookupCarriesHistory confirms that the explicit author lookup reaches the top of
// the stack carrying the document that started it.
func TestClient_AuthorLookupCarriesHistory(t *testing.T) {

	inner, root, _ := newTestStack(map[string]map[string]any{
		"https://example.com/note":  testNote("https://example.com/note", "https://example.com/alice"),
		"https://example.com/alice": testActor("https://example.com/alice"),
	})

	result, err := root.Load("https://example.com/note")
	require.NoError(t, err)

	require.Equal(t, []string{"https://example.com/note"}, root.historyFor("https://example.com/alice"))
	require.Equal(t, "Alice", result.AttributedTo().Name())
	require.Equal(t, 1, inner.loads("https://example.com/alice"))
}

// TestClient_HiddenLoadCarriesHistory confirms that a getter on a bare-URL value reaches the top of
// the stack carrying the document that holds the value.
func TestClient_HiddenLoadCarriesHistory(t *testing.T) {

	inner, root, _ := newTestStack(map[string]map[string]any{
		"https://example.com/create": {
			vocab.PropertyID:     "https://example.com/create",
			vocab.PropertyType:   vocab.ActivityTypeCreate,
			vocab.PropertyActor:  "https://example.com/alice",
			vocab.PropertyObject: "https://example.com/note",
		},
		"https://example.com/note":  testNote("https://example.com/note", "https://example.com/alice"),
		"https://example.com/alice": testActor("https://example.com/alice"),
	})

	result, err := root.Load("https://example.com/create")
	require.NoError(t, err)

	require.Equal(t, []string{"https://example.com/create"}, root.historyFor("https://example.com/note"))
	require.Equal(t, "https://example.com/note", result.ID())
	require.Equal(t, "Hello", result.Content())

	// Each getter on the bare URL loads it again, but every one stops at the normalizer's inner client
	require.Positive(t, inner.loads("https://example.com/note"))
}

// TestClient_SelfAttributedNote confirms that a Note whose author is the Note itself is loaded once,
// and its author is recorded by id.
func TestClient_SelfAttributedNote(t *testing.T) {

	// BUG-212: the author lookup re-entered the stack for the document it was normalizing.
	inner, root, _ := newTestStack(map[string]map[string]any{
		"https://example.com/note": testNote("https://example.com/note", "https://example.com/note"),
	})

	result, err := root.Load("https://example.com/note")
	require.NoError(t, err)

	require.Equal(t, 1, inner.loads("https://example.com/note"))
	require.Equal(t, "https://example.com/note", result.AttributedTo().ID())
}

// TestClient_CreateOfItself confirms that a Create whose object is its own id, as a bare URL, is
// loaded once, including by the metadata that is computed after normalizing.
func TestClient_CreateOfItself(t *testing.T) {

	// BUG-212: no explicit load is involved. Unwrapping reads the object's type, which loads it.
	inner, root, _ := newTestStack(map[string]map[string]any{
		"https://example.com/create": {
			vocab.PropertyID:     "https://example.com/create",
			vocab.PropertyType:   vocab.ActivityTypeCreate,
			vocab.PropertyActor:  "https://example.com/alice",
			vocab.PropertyObject: "https://example.com/create",
		},
	})

	result, err := root.Load("https://example.com/create")
	require.NoError(t, err)

	require.Equal(t, 1, inner.loads("https://example.com/create"))
	require.Equal(t, "https://example.com/create", result.ID())
}

// TestClient_HistoryIncludesDocumentID confirms that a document served under another URL is
// recorded by its id too, so a link back to that id stops.
func TestClient_HistoryIncludesDocumentID(t *testing.T) {

	inner, root, _ := newTestStack(map[string]map[string]any{
		"https://example.com/redirect": testNote("https://example.com/note", "https://example.com/note"),
		"https://example.com/note":     testNote("https://example.com/note", "https://example.com/note"),
	})

	_, err := root.Load("https://example.com/redirect")
	require.NoError(t, err)

	require.Equal(t, 1, inner.loads("https://example.com/redirect"))
	require.Zero(t, inner.loads("https://example.com/note"))
}

// TestClient_RepeatedURLReturnsStub confirms that a URL already in the history returns the stub
// without calling the inner client.
func TestClient_RepeatedURLReturnsStub(t *testing.T) {

	inner, _, normalizer := newTestStack(map[string]map[string]any{
		"https://example.com/note": testNote("https://example.com/note", "https://example.com/alice"),
	})

	result, err := normalizer.Load("https://example.com/note", WithHistory("https://example.com/note"))
	require.NoError(t, err)

	require.Equal(t, map[string]any{vocab.PropertyID: "https://example.com/note"}, result.Value())
	require.True(t, result.Metadata.NoStore)
	require.Zero(t, inner.totalLoads())
}

// TestClient_FragmentsAreIgnored confirms that a URL and the same URL with a fragment count as one
// document, whichever side carries the fragment.
func TestClient_FragmentsAreIgnored(t *testing.T) {

	inner, _, normalizer := newTestStack(map[string]map[string]any{
		"https://example.com/alice": testActor("https://example.com/alice"),
	})

	result, err := normalizer.Load("https://example.com/alice", WithHistory("https://example.com/alice#main-key"))
	require.NoError(t, err)
	require.Equal(t, vocab.Unknown, result.Type())

	result, err = normalizer.Load("https://example.com/alice#main-key", WithHistory("https://example.com/alice"))
	require.NoError(t, err)
	require.Equal(t, vocab.Unknown, result.Type())

	require.Zero(t, inner.totalLoads())
}

// TestClient_FreshURLChain confirms that a chain of Notes, each attributed to a new URL, is followed
// to its end with every link fetched once.
func TestClient_FreshURLChain(t *testing.T) {

	// There is no depth limit: the history stops repeats, not long chains
	const length = 20

	documents := make(map[string]map[string]any)

	for index := range length {
		documents[chainURL(index)] = testNote(chainURL(index), chainURL(index+1))
	}

	inner, root, _ := newTestStack(documents)

	_, err := root.Load(chainURL(0))
	require.NoError(t, err)

	// Every link, plus the missing one that ends the chain, is requested once
	for index := range length + 1 {
		require.Equal(t, 1, inner.loads(chainURL(index)), "link %d", index)
	}
}

// TestClient_ReturnsOriginalClient confirms that the returned document is bound to the client that
// produced it, not to the client that carries the history.
func TestClient_ReturnsOriginalClient(t *testing.T) {

	_, root, _ := newTestStack(map[string]map[string]any{
		"https://example.com/note":  testNote("https://example.com/note", "https://example.com/alice"),
		"https://example.com/alice": testActor("https://example.com/alice"),
	})

	result, err := root.Load("https://example.com/note")
	require.NoError(t, err)

	require.Equal(t, streams.Client(root), result.Client())
}

// TestClient_PassesOptionsDown confirms that a Load's own options reach the inner client unchanged.
func TestClient_PassesOptionsDown(t *testing.T) {

	inner, _, normalizer := newTestStack(map[string]map[string]any{
		"https://example.com/alice": testActor("https://example.com/alice"),
	})

	_, err := normalizer.Load("https://example.com/alice", "caller-option")
	require.NoError(t, err)

	require.Equal(t, []any{"caller-option"}, inner.optionsFor("https://example.com/alice"))
}

// TestClient_InnerErrorKeepsItsCode confirms that a failed load returns an error with its derp code
// intact.
func TestClient_InnerErrorKeepsItsCode(t *testing.T) {

	_, _, normalizer := newTestStack(nil)

	result, err := normalizer.Load("https://example.com/missing")

	require.Error(t, err)
	require.True(t, derp.IsNotFound(err))
	require.True(t, result.IsNil())
}

/******************************************
 * Helpers
 ******************************************/

// newTestStack builds a normalizer over a fakeInner, with a recordingRoot on top that passes every
// load back to the normalizer, as the top of a real stack does
func newTestStack(documents map[string]map[string]any) (*fakeInner, *recordingRoot, *Client) {

	inner := newFakeInner(documents)
	normalizer := New(inner)
	root := &recordingRoot{
		next:      normalizer,
		histories: make(map[string][]string),
	}

	normalizer.SetRootClient(root)
	return inner, root, normalizer
}

// testNote returns a Note at the given URL, attributed to the given author
func testNote(id string, attributedTo string) map[string]any {
	return map[string]any{
		vocab.PropertyID:           id,
		vocab.PropertyType:         vocab.ObjectTypeNote,
		vocab.PropertyAttributedTo: attributedTo,
		vocab.PropertyContent:      "Hello",
	}
}

// testActor returns a Person named Alice at the given URL
func testActor(id string) map[string]any {
	return map[string]any{
		vocab.PropertyID:   id,
		vocab.PropertyType: vocab.ActorTypePerson,
		vocab.PropertyName: "Alice",
	}
}

// chainURL returns the URL of one link in a chain of Notes
func chainURL(index int) string {
	return "https://example.com/chain/" + strconv.Itoa(index)
}

// recordingRoot stands in for the top of a stack: it records the history each load carries, then
// passes the load to the next client
type recordingRoot struct {
	mutex     sync.Mutex
	next      streams.Client
	histories map[string][]string
}

// Load records the history this load carries, then passes it on
func (root *recordingRoot) Load(url string, options ...any) (streams.Document, error) {

	root.mutex.Lock()
	root.histories[url] = newLoadConfig(options...).history
	root.mutex.Unlock()

	return root.next.Load(url, options...)
}

// historyFor returns the history the most recent load of this URL carried
func (root *recordingRoot) historyFor(url string) []string {
	root.mutex.Lock()
	defer root.mutex.Unlock()

	return root.histories[url]
}

// Save does nothing
func (root *recordingRoot) Save(streams.Document) error {
	return nil
}

// Delete does nothing
func (root *recordingRoot) Delete(string) error {
	return nil
}

// SetRootClient does nothing, because this is the root
func (root *recordingRoot) SetRootClient(streams.Client) {}

// fakeInner serves canned documents bound to the root client, as a real transport does, and counts
// every load by URL
type fakeInner struct {
	mutex      sync.Mutex
	rootClient streams.Client
	documents  map[string]map[string]any
	counts     map[string]int
	options    map[string][]any
}

// newFakeInner returns a fakeInner that serves the given documents and answers 404 for anything else
func newFakeInner(documents map[string]map[string]any) *fakeInner {
	return &fakeInner{
		documents: documents,
		counts:    make(map[string]int),
		options:   make(map[string][]any),
	}
}

// Load counts the request, then returns a copy of the canned document
func (inner *fakeInner) Load(url string, options ...any) (streams.Document, error) {

	const location = "asnormalizer.fakeInner.Load"

	inner.mutex.Lock()
	defer inner.mutex.Unlock()

	inner.counts[url]++
	inner.options[url] = options

	if inner.counts[url] > runawayLimit {
		return streams.NilDocument(), derp.Internal(location, "Runaway load", url)
	}

	value, found := inner.documents[url]

	if !found {
		return streams.NilDocument(), derp.NotFound(location, "No document", url)
	}

	// Each load gets its own map, as a fresh HTTP response would
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}

	return streams.NewDocument(copied, streams.WithClient(inner.rootClient)), nil
}

// loads returns how many times this URL has been requested
func (inner *fakeInner) loads(url string) int {
	inner.mutex.Lock()
	defer inner.mutex.Unlock()

	return inner.counts[url]
}

// totalLoads returns how many requests have been made for every URL
func (inner *fakeInner) totalLoads() int {
	inner.mutex.Lock()
	defer inner.mutex.Unlock()

	result := 0

	for _, count := range inner.counts {
		result += count
	}

	return result
}

// optionsFor returns the options the most recent load of this URL received
func (inner *fakeInner) optionsFor(url string) []any {
	inner.mutex.Lock()
	defer inner.mutex.Unlock()

	return inner.options[url]
}

// Save does nothing
func (inner *fakeInner) Save(streams.Document) error {
	return nil
}

// Delete does nothing
func (inner *fakeInner) Delete(string) error {
	return nil
}

// SetRootClient records the client that documents are bound to
func (inner *fakeInner) SetRootClient(rootClient streams.Client) {
	inner.mutex.Lock()
	defer inner.mutex.Unlock()

	inner.rootClient = rootClient
}
