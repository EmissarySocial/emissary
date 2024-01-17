package mastodon

import (
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
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Query the database
		ruleService := factory.Rule()
		criteria := queryExpression(t)
		rules, err := ruleService.QueryByTypeDomain(auth.UserID, criteria, option.Fields("trigger"))

		if err != nil {
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Error querying database")
		}

		// Extract *just* the domain trigger...
		result := slice.Map[model.Rule](rules, func(rule model.Rule) string {
			return rule.Trigger
		})

		return result, getPageInfo(rules), nil
	}
}

func PostDomainBlock(serverFactory *server.Factory) func(model.Authorization, txn.PostDomainBlock) (struct{}, error) {

	const location = "handler.mastodon.PostDomainBlock"

	return func(auth model.Authorization, t txn.PostDomainBlock) (struct{}, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Create the new "Domain Rule"
		rule := model.NewRule()
		rule.UserID = auth.UserID
		rule.Type = model.RuleTypeDomain
		rule.Trigger = t.Domain

		// Save it to the database
		ruleService := factory.Rule()
		if err := ruleService.Save(&rule, "Created via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error saving rule")
		}

		return struct{}{}, nil
	}
}

func DeleteDomainBlock(serverFactory *server.Factory) func(model.Authorization, txn.DeleteDomainBlock) (struct{}, error) {

	const location = "handler.mastodon.DeleteDomainRule"

	return func(auth model.Authorization, t txn.DeleteDomainBlock) (struct{}, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Try to find the Rule in the database
		ruleService := factory.Rule()
		rule := model.NewRule()

		if err := ruleService.LoadByTrigger(auth.UserID, model.RuleTypeDomain, t.Domain, &rule); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error loading rule")
		}

		// Delete the Rule from the database
		if err := ruleService.Delete(&rule, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error deleting rule")
		}

		return struct{}{}, nil
	}
}
