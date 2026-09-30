package mastodon

import (
	"strconv"
	"sync"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
	"time"
)

// https://docs.joinmastodon.org/methods/blocks/
func GetBlocks(serverFactory *server.Factory) func(model.Authorization, txn.GetBlocks) ([]object.Account, toot.PageInfo, error) {

	return func(auth model.Authorization, t txn.GetBlocks) ([]object.Account, toot.PageInfo, error) {
		return listActorRuleAccounts(serverFactory, auth, t.Host, t, t.Limit, model.RuleActionBlock, "handler.mastodon.GetBlocks")
	}
}

// listActorRuleAccounts lists the accounts a User has blocked or muted (action), newest first.
// A Rule keys on the actor's URL, so this covers remote actors too, not just local Users.
func listActorRuleAccounts(serverFactory *server.Factory, auth model.Authorization, host string, pager txn.QueryPager, limit int64, action string, location string) ([]object.Account, toot.PageInfo, error) {

	factory, session, cancel, err := statusSession(serverFactory, host, location)

	if err != nil {
		return nil, toot.PageInfo{}, err
	}

	defer cancel()

	rules, err := factory.Rule().QueryActorRulesByAction(session, auth.UserID, action, queryExpression(pager), option.MaxRows(pageLimit(limit)))

	if err != nil {
		return nil, toot.PageInfo{}, derp.Wrap(err, location, "Querying rules")
	}

	pageInfo := toot.PageInfo{}

	if length := len(rules); length > 0 {
		pageInfo.MaxID = strconv.FormatInt(rules[length-1].CreateDate, 10)
		pageInfo.MinID = strconv.FormatInt(rules[0].CreateDate, 10)
	}

	return ruleActorsToAccounts(factory, session, auth, rules), pageInfo, nil
}

// ruleActorsToAccounts turns each rule's actor address into an account, concurrently: a local
// User, else the actor as fetched (usually from cache), else a basic account built from the URL
// alone so an unreachable actor can still be found in the list and un-blocked or un-muted.
func ruleActorsToAccounts(factory *service.Factory, session data.Session, auth model.Authorization, rules []model.Rule) []object.Account {

	const maxConcurrent = 16

	accounts := make([]object.Account, len(rules))

	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, maxConcurrent)

	for index := range rules {

		waitGroup.Add(1)
		slots <- struct{}{}

		go func(index int) {
			defer waitGroup.Done()
			defer func() { <-slots }()
			accounts[index] = ruleActorToAccount(factory, session, auth, rules[index])
		}(index)
	}

	waitGroup.Wait()
	return accounts
}

// ruleActorToAccount converts one rule's actor address into an account (see ruleActorsToAccounts).
func ruleActorToAccount(factory *service.Factory, session data.Session, auth model.Authorization, rule model.Rule) object.Account {

	user := model.NewUser()

	if err := factory.User().LoadByProfileURL(session, rule.Trigger, &user); err == nil {
		return tootUser(factory, session, auth, &user)
	}

	client := factory.ActivityStream().UserClient(auth.UserID)

	if document, err := client.Load(rule.Trigger); err == nil && document.IsActor() {
		return mapDocumentToAccount(factory, session, document)
	}

	return model.RemoteActorAccount(rule.Trigger, "", "", time.UnixMilli(rule.CreateDate))
}
