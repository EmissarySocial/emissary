package mastodon

import (
	"strings"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

const (
	searchDefaultLimit = 20 // Mastodon's documented default page size for statuses and hashtags
	searchMaxLimit     = 40 // Mastodon's documented maximum
)

// https://docs.joinmastodon.org/methods/search/
//
// Searches accounts, statuses, and hashtags, or only the one named by type.
func GetSearch(serverFactory *server.Factory) func(model.Authorization, txn.GetSearch) (object.Search, error) {

	const location = "handler.mastodon_GetSearch"

	return func(auth model.Authorization, t txn.GetSearch) (object.Search, error) {

		result := object.Search{}
		query := strings.TrimSpace(t.Q)

		if query == "" {
			return result, nil
		}

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return result, err
		}

		defer cancel()

		if t.Type == "" || t.Type == "accounts" {

			accounts, err := searchAccounts(factory, session, auth, t.Host, query, t.Following, t.Limit, t.Offset)

			if err != nil {
				return result, derp.Wrap(err, location, "Searching accounts", query)
			}

			result.Accounts = accounts
		}

		if t.Type == "" || t.Type == "statuses" {

			statuses, err := searchStatuses(factory, session, auth, query, t.Limit, t.Offset)

			if err != nil {
				return result, derp.Wrap(err, location, "Searching statuses", query)
			}

			result.Statuses = statuses
		}

		return result, nil
	}
}

// searchWindow clamps a client's limit and offset to the page sizes Mastodon documents.
func searchWindow(limit int64, offset int) (int64, int) {

	if limit <= 0 {
		limit = searchDefaultLimit
	}

	return min(limit, searchMaxLimit), max(offset, 0)
}

// searchStatuses finds indexed local posts matching the query, newest first, that the caller may view.
func searchStatuses(factory *service.Factory, session data.Session, auth model.Authorization, query string, limit int64, offset int) ([]object.Status, error) {

	const location = "handler.mastodon.searchStatuses"

	limit, offset = searchWindow(limit, offset)
	wanted := int(limit) + offset

	// Accounts are indexed alongside posts, so leave them out here
	search := build.NewSearchBuilder(factory.SearchTag(), factory.SearchResult(), factory.Rule(), auth.UserID, session, exp.NotEqual("type", "Person"), query)
	rows, err := search.Top120().ByCreateDate().Reverse().Slice()

	if err != nil {
		return nil, derp.Wrap(err, location, "Querying search index", query)
	}

	statuses := make([]object.Status, 0, wanted)

	for _, row := range rows {

		if len(statuses) >= wanted {
			break
		}

		// An index row can outlive the visibility it was created under, so check the Stream itself
		stream := model.NewStream()

		if err := factory.Stream().LoadByURL(session, row.URL, &stream); err != nil {
			continue
		}

		if err := userCanStream(factory, session, &auth, &stream, "view"); err != nil {
			continue
		}

		statuses = append(statuses, tootStream(factory, session, &stream))
	}

	if offset >= len(statuses) {
		return []object.Status{}, nil
	}

	return statuses[offset:], nil
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
