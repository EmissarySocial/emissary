package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
)

// featuredMaxPosts is the most pinned posts read from one profile; servers allow only a handful.
const featuredMaxPosts = 20

// featuredPostURLs lists the addresses of the posts an actor has pinned, from its "featured" collection.
func featuredPostURLs(client streams.Client, actor streams.Document, limit int) []string {

	featured, ok := loadIfLink(client, actor.Featured())

	if !ok || featured.IsNil() {
		return []string{}
	}

	result := []string{}

	for item := range collections.RangeDocuments(featured, collections.WithMaxDocuments(limit)) {

		if id := item.ID(); id != "" {
			result = append(result, id)
		}
	}

	return result
}

// remotePostStatus builds the Status for a post by a remote author: from the caller's news feed if it
// is there, so it keeps the ID the client already knows, and straight from the post otherwise.
func remotePostStatus(client streams.Client, factory *service.Factory, session data.Session, auth model.Authorization, accounts *accountMemo, post streams.Document, account object.Account) object.Status {

	newsItem := model.NewNewsItem()

	if err := factory.NewsFeed().LoadByURL(session, auth.UserID, post.ID(), &newsItem); err == nil {
		status, _ := newsItemToStatus(client, factory, session, accounts, newsItem)
		return status
	}

	status := documentToStatus(post, account)
	applyRemoteReply(&status, client, factory, session, post)

	return status
}

// remoteFeaturedStatuses returns the posts a remote account has pinned to its profile, in the order its
// server lists them. A pinned post that cannot be read is left out, and so is anything that is not a post.
func remoteFeaturedStatuses(factory *service.Factory, session data.Session, auth model.Authorization, accountURL string) ([]object.Status, toot.PageInfo, error) {

	client := factory.ActivityStream().UserClient(auth.UserID)
	result := []object.Status{}

	actor, err := client.Load(accountURL)

	if err != nil {
		return result, toot.PageInfo{}, nil
	}

	account := mapDocumentToAccount(factory, session, actor)
	accounts := newAccountMemo()

	for _, postURL := range featuredPostURLs(client, actor, featuredMaxPosts) {

		post, err := client.Load(postURL)

		if err != nil || post.ID() == "" || (post.Content() == "" && post.Attachment().IsNil()) {
			continue
		}

		status := remotePostStatus(client, factory, session, auth, accounts, post, account)
		status.Pinned = true

		result = append(result, status)
	}

	markReacted(factory, session, auth.UserID, result)
	markBookmarked(factory, session, auth.UserID, result)

	return result, toot.PageInfo{}, nil
}
