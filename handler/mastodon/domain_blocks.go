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
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()
		// Query the database
		ruleService := factory.Rule()
		criteria := queryExpression(t)
		rules, err := ruleService.QueryByTypeDomain(session, auth.UserID, criteria, option.Fields("trigger"))

		if err != nil {
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Querying database")
		}

		// Extract *just* the domain trigger...
		result := slice.Map(rules, func(rule model.Rule) string {
			return rule.Trigger
		})

		return result, getPageInfo(rules), nil
	}
}

// PostDomainBlock implements the Mastodon "block domain" endpoint
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
			return struct{}{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// Create the new "Domain Rule"
		// RULE: A Mastodon "domain block" is a BLOCK, not NewRule()'s MUTE default
		rule := model.NewRule()
		rule.UserID = auth.UserID
		rule.Type = model.RuleTypeDomain
		rule.Action = model.RuleActionBlock
		rule.Trigger = t.Domain

		// Save it to the database
		ruleService := factory.Rule()
		if err := ruleService.Save(session, &rule, "Created via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Saving rule")
		}

		return struct{}{}, nil
	}
}

// DeleteDomainBlock implements the Mastodon "unblock domain" endpoint
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
			return struct{}{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()
		// Try to find the Rule in the database
		ruleService := factory.Rule()
		rule := model.NewRule()

		if err := ruleService.LoadByMatchKey(session, auth.UserID, model.RuleTypeDomain, t.Domain, &rule); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Loading rule")
		}

		// Delete the Rule from the database
		if err := ruleService.Delete(session, &rule, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Deleting rule")
		}

		return struct{}{}, nil
	}
}
