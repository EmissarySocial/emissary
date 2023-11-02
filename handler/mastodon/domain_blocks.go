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

	const location = "handler.mastodon.DomainBlocks"

	return func(auth model.Authorization, t txn.GetDomainBlocks) ([]string, toot.PageInfo, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Query the database
		blockService := factory.Block()
		criteria := queryExpression(t)
		blocks, err := blockService.QueryByTypeDomain(auth.UserID, criteria, option.Fields("trigger"))

		if err != nil {
			return []string{}, toot.PageInfo{}, derp.Wrap(err, location, "Error querying database")
		}

		// Extract *just* the domain trigger...
		result := slice.Map[model.Block](blocks, func(block model.Block) string {
			return block.Trigger
		})

		return result, getPageInfo(blocks), nil
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

		// Create the new "Domain Block"
		block := model.NewBlock()
		block.UserID = auth.UserID
		block.Type = model.BlockTypeDomain
		block.Trigger = t.Domain
		block.IsActive = true

		// Save it to the database
		blockService := factory.Block()
		if err := blockService.Save(&block, "Created via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error saving block")
		}

		return struct{}{}, nil
	}
}

func DeleteDomainBlock(serverFactory *server.Factory) func(model.Authorization, txn.DeleteDomainBlock) (struct{}, error) {

	const location = "handler.mastodon.DeleteDomainBlock"

	return func(auth model.Authorization, t txn.DeleteDomainBlock) (struct{}, error) {

		// Get the Domain factory for this request
		factory, err := serverFactory.ByDomainName(t.Host)

		if err != nil {
			return struct{}{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Try to find the Block in the database
		blockService := factory.Block()
		block := model.NewBlock()

		if err := blockService.LoadByTrigger(auth.UserID, model.BlockTypeDomain, t.Domain, &block); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error loading block")
		}

		// Delete the Block from the database
		if err := blockService.Delete(&block, "Deleted via Mastodon API"); err != nil {
			return struct{}{}, derp.Wrap(err, location, "Error deleting block")
		}

		return struct{}{}, nil
	}
}
