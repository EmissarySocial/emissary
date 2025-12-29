package mastodon

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/toot"
	"github.com/benpate/toot/txn"
)

// https://docs.joinmastodon.org/methods/domain_blocks/
func GetDomainBlocks(serverFactory *server.Factory) func(model.Authorization, txn.GetDomainBlocks) ([]string, toot.PageInfo, error) {

	const location = "handler.mastodon.DomainRules"

	return func(auth model.Authorization, t txn.GetDomainBlocks) ([]string, toot.PageInfo, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()
		// Query the database
		ruleService := factory.Rule()
		criteria := queryExpression(t)
		rules, err := ruleService.QueryByTypeDomain(session, auth.UserID, criteria, option.Fields("trigger"))

		if err != nil {
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Error querying database")
		}

		// Extract *just* the domain trigger...
		result := slice.Map(rules, func(rule model.Rule) string {
			return rule.Trigger
		})

		return result, getPageInfo(rules), nil
	}
}

func PostDomainBlock(serverFactory *server.Factory) func(model.Authorization, txn.PostDomainBlock) (struct{}, error) {

	const location = "handler.mastodon.PostDomainBlock"

	return func(auth model.Authorization, t txn.PostDomainBlock) (struct{}, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()
		// Create the new "Domain Rule"
		rule := model.NewRule()
		rule.UserID = auth.UserID
		rule.Type = model.RuleTypeDomain
		rule.Trigger = t.Domain

		// Save it to the database
		ruleService := factory.Rule()
		if err := ruleService.Save(session, &rule, "Created via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unable to save rule")
		}

		return struct{}{}, nil
	}
}

func DeleteDomainBlock(serverFactory *server.Factory) func(model.Authorization, txn.DeleteDomainBlock) (struct{}, error) {

	const location = "handler.mastodon.DeleteDomainRule"

	return func(auth model.Authorization, t txn.DeleteDomainBlock) (struct{}, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unable to create session")
		}

		defer cancel()
		// Try to find the Rule in the database
		ruleService := factory.Rule()
		rule := model.NewRule()

		if err := ruleService.LoadByTrigger(session, auth.UserID, model.RuleTypeDomain, t.Domain, &rule); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unable to load rule")
		}

		// Delete the Rule from the database
		if err := ruleService.Delete(session, &rule, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unable to delete rule")
		}

		return struct{}{}, nil
	}
}
