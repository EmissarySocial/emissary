package mastodon

import (
	"encoding/base64"
	"net/url"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
)

// remoteListMaxPages caps how many collection pages one request will read.
const remoteListMaxPages = 5

// remoteListWorkers caps how many actors one request will fetch at the same time.
const remoteListWorkers = 8

// remoteAccountList reads a page of accounts from a remote actor's followers or following collection.
// The cursor is an opaque token for the next collection page, as returned in the PageInfo.
func remoteAccountList(client streams.Client, factory *service.Factory, session data.Session, collection streams.Document, cursor string, limit int64) ([]object.Account, toot.PageInfo) {

	actorURLs, nextURL := collectActorURLs(client, collection, cursor, limit)

	pageInfo := toot.PageInfo{}

	if nextURL != "" {
		pageInfo.MaxID = base64.RawURLEncoding.EncodeToString([]byte(nextURL))
	}

	return loadRemoteAccounts(client, factory, session, actorURLs), pageInfo
}

// collectActorURLs walks a collection's pages from the cursor, gathering actor URLs until there are
// enough. It also returns the next page's URL, or "" when the collection is exhausted.
func collectActorURLs(client streams.Client, collection streams.Document, cursor string, limit int64) ([]string, string) {

	actorURLs := make([]string, 0, limit)

	// Load the collection if it is only a link
	collection, ok := loadIfLink(client, collection)

	if !ok {
		return actorURLs, ""
	}

	// Find the page to start from
	page, ok := firstCollectionPage(client, collection, cursor)

	if !ok {
		return actorURLs, ""
	}

	// Gather actor URLs page by page until there are enough
	nextURL := ""

	for pageCount := 0; pageCount < remoteListMaxPages; pageCount++ {

		for item := range page.Items().Range() {
			if id := item.ID(); id != "" {
				actorURLs = append(actorURLs, id)
			}
		}

		// Stop when there is no next page, or enough actors
		nextURL = page.Next().ID()

		if nextURL == "" || int64(len(actorURLs)) >= limit {
			break
		}

		// Load the next page
		next, err := client.Load(nextURL)

		if err != nil {
			return actorURLs, ""
		}

		page = next
	}

	return actorURLs, nextURL
}

// firstCollectionPage returns the collection page to start reading from: the page named by the
// cursor, or else the collection's first page (or the collection itself when it holds items inline).
func firstCollectionPage(client streams.Client, collection streams.Document, cursor string) (streams.Document, bool) {

	if collection.IsNil() {
		return streams.NilDocument(), false
	}

	if cursor != "" {

		raw, err := base64.RawURLEncoding.DecodeString(cursor)

		if err != nil {
			return streams.NilDocument(), false
		}

		// RULE: a cursor may only point back into the same server's collection, never at a different host.
		if !sameHost(string(raw), collection.ID()) {
			return streams.NilDocument(), false
		}

		page, err := client.Load(string(raw))
		return page, err == nil
	}

	first := collection.First()

	if first.IsNil() {
		return collection, true
	}

	return loadIfLink(client, first)
}

// loadIfLink loads a document that is only a link (a bare URL), and passes through one that is already loaded.
func loadIfLink(client streams.Client, document streams.Document) (streams.Document, bool) {

	if !document.IsString() {
		return document, true
	}

	loaded, err := client.Load(document.ID())
	return loaded, err == nil
}

// sameHost returns TRUE if both addresses are web URLs on the same host.
func sameHost(left string, right string) bool {

	leftURL, err := url.Parse(left)

	if err != nil || (leftURL.Scheme != "https" && leftURL.Scheme != "http") || leftURL.Host == "" {
		return false
	}

	rightURL, err := url.Parse(right)

	if err != nil {
		return false
	}

	return leftURL.Host == rightURL.Host
}

// loadRemoteAccounts builds an account for each actor URL, in order. An actor that cannot be
// loaded is shown from what its URL alone tells us, so the list never silently drops people.
func loadRemoteAccounts(client streams.Client, factory *service.Factory, session data.Session, actorURLs []string) []object.Account {

	documents := make([]streams.Document, len(actorURLs))
	loaded := make([]bool, len(actorURLs))

	// Fetch actors (and their counts, to warm the cache) in parallel
	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, remoteListWorkers)

	for index, actorURL := range actorURLs {

		waitGroup.Add(1)
		slots <- struct{}{}

		go func(index int, actorURL string) {
			defer waitGroup.Done()
			defer func() { <-slots }()

			document, err := client.Load(actorURL)

			if err != nil {
				return
			}

			documents[index] = document
			loaded[index] = true
			document.Followers().LoadLink().TotalItems()
			document.Following().LoadLink().TotalItems()
			document.Outbox().LoadLink().TotalItems()
		}(index, actorURL)
	}

	waitGroup.Wait()

	// Map them one at a time, because mapping uses the database session
	result := make([]object.Account, len(actorURLs))

	for index, actorURL := range actorURLs {

		if !loaded[index] {
			result[index] = model.RemoteActorAccount(actorURL, "", "", time.Time{})
			continue
		}

		result[index] = mapDocumentToAccount(factory, session, documents[index])
	}

	return result
}
