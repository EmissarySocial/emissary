package mastodon

import (
	"strconv"
	"sync"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
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

// actorRef names an actor by URL, with the date to show if it can only be listed by that URL.
type actorRef struct {
	url   string
	since time.Time
}

// ruleActorsToAccounts turns each rule's actor address into an account (see actorsToAccounts), so an
// unreachable actor can still be found in the list and un-blocked or un-muted.
func ruleActorsToAccounts(factory *service.Factory, session data.Session, auth model.Authorization, rules []model.Rule) []object.Account {

	actors := make([]actorRef, len(rules))

	for index, rule := range rules {
		actors[index] = actorRef{url: rule.Trigger, since: time.UnixMilli(rule.CreateDate)}
	}

	return actorsToAccounts(factory, session, auth, actors)
}

// actorsToAccounts turns each actor into an account, concurrently: a local User, else the actor
// as fetched (usually from cache), else a basic account built from the URL alone.
func actorsToAccounts(factory *service.Factory, session data.Session, auth model.Authorization, actors []actorRef) []object.Account {

	const maxConcurrent = 16

	accounts := make([]object.Account, len(actors))

	var waitGroup sync.WaitGroup
	slots := make(chan struct{}, maxConcurrent)

	for index := range actors {

		waitGroup.Add(1)
		slots <- struct{}{}

		go func(index int) {
			defer waitGroup.Done()
			defer func() { <-slots }()
			accounts[index] = actorToAccount(factory, session, auth, actors[index])
		}(index)
	}

	waitGroup.Wait()
	return accounts
}

// actorToAccount converts one actor into an account (see actorsToAccounts).
func actorToAccount(factory *service.Factory, session data.Session, auth model.Authorization, actor actorRef) object.Account {

	user := model.NewUser()

	if err := factory.User().LoadByProfileURL(session, actor.url, &user); err == nil {
		return tootUser(factory, session, auth, &user)
	}

	client := factory.ActivityStream().UserClient(auth.UserID)

	if document, err := client.Load(actor.url); err == nil && document.IsActor() {
		return mapDocumentToAccount(factory, session, document)
	}

	return model.RemoteActorAccount(actor.url, "", "", actor.since)
}
