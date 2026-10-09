package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/toot/object"
)

// documentAuthorURL returns the address of a post's author, taken from attributedTo or else actor.
func documentAuthorURL(document streams.Document) string {

	if authorURL := document.AttributedTo().ID(); authorURL != "" {
		return authorURL
	}

	return document.ActorID()
}

// remoteReplyIDs returns the status ID and the author's account ID of the post a document replies to,
// or the status ID alone when the parent cannot be read.
func remoteReplyIDs(client streams.Client, factory *service.Factory, session data.Session, document streams.Document) (string, string) {

	parentURL := document.InReplyTo().ID()

	if parentURL == "" {
		return "", ""
	}

	// A parent on this server is a Stream, whose author is already known
	parent := model.NewStream()

	if err := factory.Stream().LoadByURL(session, parentURL, &parent); err == nil {
		return replyIDsFor(parentURL, &parent)
	}

	// Any other parent has to be read from its own server to learn who wrote it
	parentDocument, err := client.Load(parentURL)

	// RULE: clients show "replying to" only when given both IDs, so a missing author hides the line
	if err != nil {
		return model.EncodeRemoteStatusID(parentURL), ""
	}

	authorURL := documentAuthorURL(parentDocument)

	if authorURL == "" {
		return model.EncodeRemoteStatusID(parentURL), ""
	}

	return model.EncodeRemoteStatusID(parentURL), resolveAccountID(factory, session, authorURL)
}

// applyRemoteReply fills in what a post from another server replies to.
func applyRemoteReply(status *object.Status, client streams.Client, factory *service.Factory, session data.Session, document streams.Document) {
	status.InReplyToID, status.InReplyToAccountID = remoteReplyIDs(client, factory, session, document)
}

// accountForActor returns the account for an author's address: a User here, else the actor as read from
// its own server, else a basic account. Each address is looked up once per request.
func accountForActor(client streams.Client, factory *service.Factory, session data.Session, auth model.Authorization, authorURL string, accounts *accountMemo) object.Account {

	account, _ := accounts.get(authorURL, func() (object.Account, bool) {

		if user, err := loadUserByAccountID(factory, session, authorURL); err == nil {
			return tootUser(factory, session, auth, &user), true
		}

		if loaded, found := loadAccount(client, factory, session, authorURL); found {
			return loaded, true
		}

		return model.RemoteActorAccount(authorURL, "", "", time.Time{}), true
	})

	return account
}

// walkThread follows a chain of parents upward from a starting address, calling step for each post, and
// returns the Statuses oldest first. It stops at an unusable post, a repeat, or the limit.
func walkThread(startURL string, limit int, step func(url string) (object.Status, string, bool)) []object.Status {

	result := []object.Status{}
	seen := map[string]bool{}

	for url := startURL; url != "" && !seen[url] && len(result) < limit; {

		seen[url] = true

		// A step gives the post's Status and its parent's address, or false if the post can't be used
		status, next, ok := step(url)

		if !ok {
			break
		}

		result = append([]object.Status{status}, result...)
		url = next
	}

	return result
}

// remoteAncestorsOf lists a post and the posts above it, oldest first, from the parent's address. Each
// comes from the database if it lives here, and from its own server otherwise.
func remoteAncestorsOf(client streams.Client, factory *service.Factory, session data.Session, auth model.Authorization, parentURL string) []object.Status {

	accounts := newAccountMemo()

	result := walkThread(parentURL, contextMaxAncestors, func(url string) (object.Status, string, bool) {

		// A post on this server comes from the database, if the caller may view it
		stream := model.NewStream()

		if err := factory.Stream().LoadByURL(session, url, &stream); err == nil {

			if err := userCanStream(factory, session, &auth, &stream, "view"); err != nil {
				return object.Status{}, "", false
			}

			return tootStream(factory, session, &stream), stream.InReplyTo, true
		}

		// Any other post is read from its own server
		document, err := client.Load(url)

		if err != nil {
			return object.Status{}, "", false
		}

		status := documentToStatus(document, accountForActor(client, factory, session, auth, documentAuthorURL(document), accounts))
		applyRemoteReply(&status, client, factory, session, document)

		return status, document.InReplyTo().ID(), true
	})

	markReacted(factory, session, auth.UserID, result)
	markBookmarked(factory, session, auth.UserID, result)

	return result
}
