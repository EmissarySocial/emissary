package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

const (
	directoryDefaultLimit = 40 // Mastodon's documented default page size
	directoryMaxLimit     = 80 // Mastodon's documented maximum
)

// https://docs.joinmastodon.org/methods/directory/
//
// Lists this server's public accounts, newest first for order=new and most recently updated
// first otherwise. Emissary has no other servers' accounts to list, so remote is not honored.
func GetDirectory(serverFactory *server.Factory) func(model.Authorization, txn.GetDirectory) ([]object.Account, toot.PageInfo, error) {

	const location = "handler.mastodon.GetDirectory"

	return func(auth model.Authorization, t txn.GetDirectory) ([]object.Account, toot.PageInfo, error) {

		factory, session, cancel, err := statusSession(serverFactory, t.Host, location)

		if err != nil {
			return nil, toot.PageInfo{}, err
		}

		defer cancel()

		limit := t.Limit

		if limit <= 0 {
			limit = directoryDefaultLimit
		}

		limit = min(limit, directoryMaxLimit)
		offset := max(t.Offset, 0)

		sortField := "updateDate"

		if t.Order == "new" {
			sortField = "createDate"
		}

		// The data layer has no offset option, so read through the offset and skip it here
		users, err := factory.User().Query(session, exp.Equal("isPublic", true), option.SortDesc(sortField), option.MaxRows(int64(offset+limit)))

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying users")
		}

		if offset >= len(users) {
			return []object.Account{}, toot.PageInfo{}, nil
		}

		users = users[offset:]

		accounts := make([]object.Account, len(users))

		for index := range users {
			accounts[index] = tootUser(factory, session, auth, &users[index])
		}

		return accounts, toot.PageInfo{}, nil
	}
}
