package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

const (
	contextMaxAncestors   = 20 // Most earlier posts shown above a post
	contextMaxDescendants = 60 // Most replies shown below a post
	contextMaxDepth       = 20 // Deepest chain of replies followed
)

// GetStatus_Context implements the Mastodon "get thread" endpoint: the posts above and below a post.
// https://docs.joinmastodon.org/methods/statuses/#context
func GetStatus_Context(serverFactory *server.Factory) func(model.Authorization, txn.GetStatus_Context) (object.Context, error) {

	const location = "handler.mastodon.GetStatus_Context"

	return func(auth model.Authorization, t txn.GetStatus_Context) (object.Context, error) {

		// RULE: nil slices marshal to JSON `null`, but the Mastodon client's Codable decoder
		// requires a real (even empty) array for both fields.
		result := object.Context{
			Ancestors:   []object.Status{},
			Descendants: []object.Status{},
		}

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return result, err
		}

		defer cancel()

		client := factory.ActivityStream().UserClient(auth.UserID)

		// A post from another server shows what it replies to
		stream := model.NewStream()

		if err := loadStreamByStatusID(factory, session, t.ID, &stream); err != nil {

			if postURL := resolveStatusURL(factory, session, auth, t.ID); webURL(postURL) {

				if post, err := client.Load(postURL); err == nil {
					result.Ancestors = remoteAncestorsOf(client, factory, session, auth, post.InReplyTo().ID())
				}
			}

			return result, nil
		}

		if err := userCanStream(factory, session, &auth, &stream, "view"); err != nil {
			return result, derp.Wrap(err, location, "Viewing stream", t.ID)
		}

		parentOf := func(url string) *model.Stream {

			parent := model.NewStream()

			if err := factory.Stream().LoadByURL(session, url, &parent); err != nil {
				return nil
			}

			if err := userCanStream(factory, session, &auth, &parent, "view"); err != nil {
				return nil
			}

			return &parent
		}

		repliesOf := func(url string) []model.Stream {

			replies, err := factory.Stream().QueryReplies(session, auth, url)

			if err != nil {
				derp.Report(derp.Wrap(err, location, "Querying replies", url))
				return nil
			}

			return replies
		}

		ancestors := walkAncestors(&stream, parentOf, contextMaxAncestors)
		descendants := flattenReplies(stream.URL, repliesOf, contextMaxDescendants, contextMaxDepth)

		result.Ancestors = streamsToStatuses(factory, session, auth, ancestors)

		// When the thread climbs out of this server, carry on upward on the other one
		top := &stream

		if len(ancestors) > 0 {
			top = &ancestors[0]
		}

		if top.InReplyTo != "" && len(ancestors) < contextMaxAncestors {
			result.Ancestors = append(remoteAncestorsOf(client, factory, session, auth, top.InReplyTo), result.Ancestors...)
		}

		result.Descendants = streamsToStatuses(factory, session, auth, descendants)

		return result, nil
	}
}

// walkAncestors follows a post's chain of parents upward, returning them root first. It stops at
// the first parent that isn't available, and never visits the same post twice.
func walkAncestors(start *model.Stream, parentOf func(url string) *model.Stream, limit int) []model.Stream {

	result := []model.Stream{}

	for seen, current := map[string]bool{start.URL: true}, start; len(result) < limit && current.InReplyTo != "" && !seen[current.InReplyTo]; {

		parent := parentOf(current.InReplyTo)

		if parent == nil {
			break
		}

		seen[parent.URL] = true
		result = append([]model.Stream{*parent}, result...)
		current = parent
	}

	return result
}

// flattenReplies lists the replies under a post in thread order: each reply is followed by its own
// replies, oldest first. It stops at the limit and the maximum depth, and never visits a post twice.
func flattenReplies(parentURL string, repliesOf func(url string) []model.Stream, limit int, maxDepth int) []model.Stream {

	result := []model.Stream{}
	seen := map[string]bool{parentURL: true}

	var walk func(url string, depth int)

	walk = func(url string, depth int) {

		if depth > maxDepth {
			return
		}

		for _, reply := range repliesOf(url) {

			if len(result) >= limit {
				return
			}

			if seen[reply.URL] {
				continue
			}

			seen[reply.URL] = true
			result = append(result, reply)
			walk(reply.URL, depth+1)
		}
	}

	walk(parentURL, 1)
	return result
}

// streamsToStatuses converts posts into Statuses carrying the caller's own reaction and bookmark state.
func streamsToStatuses(factory *service.Factory, session data.Session, auth model.Authorization, streams []model.Stream) []object.Status {

	statuses := make([]object.Status, len(streams))

	for index := range streams {
		statuses[index] = tootStream(factory, session, &streams[index])
	}

	markReacted(factory, session, auth.UserID, statuses)
	markBookmarked(factory, session, auth.UserID, statuses)

	return statuses
}
