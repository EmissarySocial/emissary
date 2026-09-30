package mastodon

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/mutes/
func GetMutes(serverFactory *server.Factory) func(model.Authorization, txn.GetMutes) ([]object.Account, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetMutes) ([]object.Account, toot.PageInfo, error) {
		return listActorRuleAccounts(serverFactory, auth, t.Host, t, t.Limit, model.RuleActionMute, "handler.mastodon.GetMutes")
	}
}
