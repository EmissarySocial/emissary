package mastodon

import (
	"strconv"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/toot"
	"github.com/benpate/toot/object"
	"github.com/benpate/toot/txn"
)

// GetTrends implements Mastodon's older trending-tags endpoint, listing this server's featured tags
// https://docs.joinmastodon.org/methods/trends/#tags
func GetTrends(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends) ([]object.Tag, toot.PageInfo, error) {

	// The official Mastodon iOS app still reads its Hashtags tab from this address, so it matches GetTrends_Tags

	const location = "handler.mastodon.GetTrends"

	return func(auth model.Authorization, t txn.GetTrends) ([]object.Tag, toot.PageInfo, error) {

		// RULE: the featured list is one short page, so a later page is empty rather than a repeat
		if t.Offset > 0 {
			return []object.Tag{}, toot.PageInfo{}, nil
		}

		// Get the Domain factory for this request
		factory, err := serverFactory.ByHostname(t.Host)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Get a database session for this request
		session, cancel, err := factory.Session(time.Minute)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Creating session")
		}

		defer cancel()

		// List the featured tags
		limit := trendingTagsLimit(strconv.Itoa(t.Limit))
		result, err := trendingTags(factory, session, limit)

		if err != nil {
			return nil, toot.PageInfo{}, derp.Wrap(err, location, "Listing trending tags")
		}

		// Trendy
		return result, toot.PageInfo{}, nil
	}
}

// GetTrends_Statuses implements the Mastodon "trending statuses" endpoint, and always returns an empty list
func GetTrends_Statuses(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends_Statuses) ([]object.Status, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetTrends_Statuses) ([]object.Status, toot.PageInfo, error) {
		return []object.Status{}, toot.PageInfo{}, nil
	}
}

// GetTrends_Links implements the Mastodon "trending links" endpoint, and always returns an empty list
func GetTrends_Links(serverFactory *server.Factory) func(model.Authorization, txn.GetTrends_Links) ([]object.PreviewCard, toot.PageInfo, error) {

	return func(model.Authorization, txn.GetTrends_Links) ([]object.PreviewCard, toot.PageInfo, error) {
		return []object.PreviewCard{}, toot.PageInfo{}, nil
	}
}
