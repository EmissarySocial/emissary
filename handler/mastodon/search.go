package mastodon

import (
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/search/
//
// Only accounts are handled, via searchAccounts. Any other type returns an empty result.
func GetSearch(serverFactory *server.Factory) func(model.Authorization, txn.GetSearch) (object.Search, error) {

	const location = "handler.mastodon_GetSearch"

	return func(auth model.Authorization, t txn.GetSearch) (object.Search, error) {

		result := object.Search{}

		// Bail unless the caller wants accounts.
		if strings.TrimSpace(t.Q) == "" || (t.Type != "" && t.Type != "accounts") {
			return result, nil
		}

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return result, err
		}

		defer cancel()

		accounts, err := searchAccounts(factory, session, auth, t.Host, t.Q, t.Following, t.Limit, t.Offset)

		if err != nil {
			return result, derp.Wrap(err, location, "Searching accounts", t.Q)
		}

		result.Accounts = accounts
		return result, nil
	}
}

// resolveOneAccount looks up a single account by exact webfinger handle (user@domain) or actor
// URL -- a local User first, then a remote actor dereferenced live. found is false for anything
// else, including a partial/type-ahead string.
func resolveOneAccount(factory *service.Factory, session data.Session, auth model.Authorization, host string, query string) (object.Account, bool) {

	query = strings.TrimPrefix(query, "@")

	isHandle := strings.Count(query, "@") == 1 && !strings.ContainsAny(query, " \t")
	isURL := strings.HasPrefix(query, "https://") || strings.HasPrefix(query, "http://")

	if !isHandle && !isURL {
		return object.Account{}, false
	}

	// A local user -- only when the query is a bare username or an explicit
	// "username@<this domain>", so a remote "name@elsewhere" can never collide
	// with a local account that happens to share the username.
	localName := ""

	if name, domain, found := strings.Cut(query, "@"); !found {
		localName = query
	} else if domain == host {
		localName = name
	}

	if localName != "" {
		user := model.NewUser()
		if err := factory.User().LoadByUsername(session, localName, &user); err == nil {
			return tootUser(factory, session, auth, &user), true
		}
	}

	// A remote actor: dereference it. A miss is "no results", not an error.
	client := factory.ActivityStream().UserClient(auth.UserID)
	document, err := client.Load(query)

	if err != nil || !document.IsActor() {
		return object.Account{}, false
	}

	return mapDocumentToAccount(factory, session, document), true
}
