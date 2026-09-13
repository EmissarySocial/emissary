package mastodon

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// GetSearch implements a minimal slice of https://docs.joinmastodon.org/methods/search/
//
// Only the "resolve one account" case is handled: when q is a webfinger handle
// (user@domain) or an actor URL and the caller wants accounts, the actor is
// looked up (local User first, then dereferenced remotely) and returned. This is
// the one path the official client needs to open a profile it does not already
// have on screen. Full-text search over local statuses/hashtags is not
// supported, so every other query returns an empty -- but well-formed -- result.
func GetSearch(serverFactory *server.Factory) func(model.Authorization, txn.GetSearch) (object.Search, error) {

	const location = "handler.mastodon_GetSearch"

	return func(auth model.Authorization, t txn.GetSearch) (object.Search, error) {

		result := object.Search{}

		query := strings.TrimSpace(t.Q)
		query = strings.TrimPrefix(query, "@")

		// Bail unless the caller wants accounts and gave us something that could
		// name exactly one -- a handle or a URL, not a partial type-ahead string.
		if query == "" || (t.Type != "" && t.Type != "accounts") {
			return result, nil
		}

		isHandle := strings.Count(query, "@") == 1 && !strings.ContainsAny(query, " \t")
		isURL := strings.HasPrefix(query, "https://") || strings.HasPrefix(query, "http://")

		if !isHandle && !isURL {
			return result, nil
		}

		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return result, derp.Wrap(err, location, "Unrecognized Domain")
		}

		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return result, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// A local user -- only when the query is a bare username or an explicit
		// "username@<this domain>", so a remote "name@elsewhere" can never collide
		// with a local account that happens to share the username.
		localName := ""

		if name, domain, found := strings.Cut(query, "@"); !found {
			localName = query
		} else if domain == t.Host {
			localName = name
		}

		if localName != "" {
			user := model.NewUser()
			if err := factory.User().LoadByUsername(session, localName, &user); err == nil {
				result.Accounts = append(result.Accounts, user.Toot())
				return result, nil
			}
		}

		// A remote actor: dereference it. A miss is "no results", not an error.
		client := factory.ActivityStream().UserClient(auth.UserID)
		document, err := client.Load(query)

		if err != nil {
			return result, nil
		}

		if document.IsActor() {
			result.Accounts = append(result.Accounts, mapDocumentToAccount(factory, session, document))
		}

		return result, nil
	}
}
