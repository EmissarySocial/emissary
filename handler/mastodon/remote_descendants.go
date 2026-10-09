package mastodon

import (
	"sort"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/asnormalizer"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/toot/object"
)

const (
	threadMaxCandidates = 100 // Most posts read from another server to build one thread
	threadReplyDepth    = 4   // Deepest level of replies read when a server publishes no conversation
	threadLoadWorkers   = 8   // Posts fetched at the same time
)

// threadNode is one post in a thread, with a way to build its Status once it is chosen.
type threadNode struct {
	URL       string
	Published int64
	Build     func() object.Status
}

// flattenThread lists the posts under a root in thread order: each post is followed by its own
// replies, oldest first. It stops at the limit and the maximum depth, and never visits a post twice.
func flattenThread(rootURL string, childrenOf func(url string) []threadNode, limit int, maxDepth int) []threadNode {

	result := []threadNode{}
	seen := map[string]bool{rootURL: true}

	var walk func(url string, depth int)

	walk = func(url string, depth int) {

		// RULE: stop at the maximum depth
		if depth > maxDepth {
			return
		}

		// Order this post's replies oldest first
		children := childrenOf(url)

		sort.SliceStable(children, func(i int, j int) bool {
			return children[i].Published < children[j].Published
		})

		// Add each reply not yet seen, followed by its own replies
		for _, child := range children {

			if len(result) >= limit {
				return
			}

			if seen[child.URL] {
				continue
			}

			seen[child.URL] = true
			result = append(result, child)
			walk(child.URL, depth+1)
		}
	}

	// Walk the thread down from its root
	walk(rootURL, 1)
	return result
}

// threadCandidates lists the addresses of posts that may belong to a post's thread: those in the
// conversation the post's server publishes, or else those found by following its replies.
func threadCandidates(client streams.Client, post streams.Document) []string {

	if urls := conversationCandidates(client, post); len(urls) > 1 {
		return urls
	}

	return replyCandidates(client, post)
}

// conversationCandidates reads the collection a post's "context" points to, which lists every post in the conversation.
func conversationCandidates(client streams.Client, post streams.Document) []string {

	address := asnormalizer.Context(post)

	if !webURL(address) {
		return nil
	}

	collection, err := client.Load(address)

	if err != nil {
		return nil
	}

	result := []string{}

	for item := range collections.RangeDocuments(collection, collections.WithMaxDocuments(threadMaxCandidates)) {
		if id := item.ID(); id != "" {
			result = append(result, id)
		}
	}

	return result
}

// replyCandidates follows a post's replies collection, then each reply's own, a few levels deep.
func replyCandidates(client streams.Client, post streams.Document) []string {

	type level struct {
		document streams.Document
		depth    int
	}

	result := []string{}

	for seen, queue := map[string]bool{post.ID(): true}, []level{{post, 1}}; len(queue) > 0 && len(result) < threadMaxCandidates; {

		// Take the next post whose replies are still to be read
		current := queue[0]
		queue = queue[1:]

		// Load its replies collection
		replies, ok := loadIfLink(client, current.document.Replies())

		if !ok || replies.IsNil() {
			continue
		}

		// Note each new reply, and queue it so its own replies are read too
		for item := range collections.RangeDocuments(replies, collections.WithMaxDocuments(threadMaxCandidates)) {

			id := item.ID()

			if id == "" || seen[id] || len(result) >= threadMaxCandidates {
				continue
			}

			seen[id] = true
			result = append(result, id)

			if current.depth < threadReplyDepth {
				if child, err := client.Load(id); err == nil {
					queue = append(queue, level{child, current.depth + 1})
				}
			}
		}
	}

	return result
}

// loadThreadDocuments fetches each address in parallel, leaving out any that cannot be read.
func loadThreadDocuments(client streams.Client, urls []string) map[string]streams.Document {

	documents := make([]streams.Document, len(urls))
	loaded := make([]bool, len(urls))

	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, threadLoadWorkers)

	for index, url := range urls {

		waitGroup.Add(1)
		slots <- struct{}{}

		go func(index int, url string) {
			defer waitGroup.Done()
			defer func() { <-slots }()

			if document, err := client.Load(url); err == nil {
				documents[index] = document
				loaded[index] = true
			}
		}(index, url)
	}

	waitGroup.Wait()

	result := make(map[string]streams.Document, len(urls))

	for index, url := range urls {
		if loaded[index] {
			result[url] = documents[index]
		}
	}

	return result
}

// remoteDescendantsOf lists the replies under a post from another server, in thread order, including
// replies written here. Only posts that connect back to the post are kept.
func remoteDescendantsOf(client streams.Client, factory *service.Factory, session data.Session, auth model.Authorization, post streams.Document) []object.Status {

	rootURL := post.ID()
	documents := loadThreadDocuments(client, threadCandidates(client, post))
	accounts := newAccountMemo()

	// Group the posts read from other servers under the post each one replies to
	remoteChildren := map[string][]threadNode{}

	for url, document := range documents {

		parentURL := document.InReplyTo().ID()

		if parentURL == "" {
			continue
		}

		remoteChildren[parentURL] = append(remoteChildren[parentURL], threadNode{
			URL:       url,
			Published: document.Published().Unix(),
			Build: func() object.Status {
				status := documentToStatus(document, accountForActor(client, factory, session, auth, documentAuthorURL(document), accounts))
				applyRemoteReply(&status, client, factory, session, document)
				return status
			},
		})
	}

	// Replies written on this server sit alongside them
	childrenOf := func(url string) []threadNode {

		nodes := append([]threadNode{}, remoteChildren[url]...)

		replies, err := factory.Stream().QueryReplies(session, auth, url)

		if err != nil {
			return nodes
		}

		for index := range replies {

			stream := replies[index]

			nodes = append(nodes, threadNode{
				URL:       stream.ActivityPubURL(),
				Published: stream.PublishDate,
				Build:     func() object.Status { return tootStream(factory, session, &stream) },
			})
		}

		return nodes
	}

	chosen := flattenThread(rootURL, childrenOf, contextMaxDescendants, contextMaxDepth)

	// Build the chosen posts in parallel, keeping their order
	statuses := make([]object.Status, len(chosen))

	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, threadLoadWorkers)

	for index := range chosen {

		waitGroup.Add(1)
		slots <- struct{}{}

		go func(index int) {
			defer waitGroup.Done()
			defer func() { <-slots }()
			statuses[index] = chosen[index].Build()
		}(index)
	}

	waitGroup.Wait()

	markReacted(factory, session, auth.UserID, statuses)
	markBookmarked(factory, session, auth.UserID, statuses)

	return statuses
}
