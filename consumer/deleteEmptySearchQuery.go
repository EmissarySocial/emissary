package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

// DeleteEmptySearchQuery deletes the searchQuery IF is has no followers
func DeleteEmptySearchQuery(factory *domain.Factory, args mapof.Any) queue.Result {
	const location = "consumer.DeleteEmptySearchQuery"

	// Try to find the existing SearchQuery
	searchQueryService := factory.SearchQuery()
	token := args.GetString("searchQueryID")
	searchQuery := model.NewSearchQuery()

	if err := searchQueryService.LoadByToken(token, &searchQuery); err != nil {

		if derp.IsNotFound(err) {
			return queue.Success()
		}

		return queue.Error(derp.Wrap(err, location, "Error locating searchQuery", args))
	}

	// Count the number of Followers that this SearchQuery has
	followerService := factory.Follower()
	followerCount, err := followerService.CountByParent(model.FollowerTypeSearch, searchQuery.SearchQueryID)

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error counting followers", args))
	}

	// If the SearchQuery still has followers, then there's nothing to do.  Exit in peace.
	if followerCount > 0 {
		return queue.Success()
	}

	// Otherwise, the SearchQuery has no followers, so delete it
	if err := searchQueryService.Delete(&searchQuery, "SearchQuery has no followers"); err != nil {
		return queue.Error(derp.Wrap(err, location, "Error deleting searchQuery", args))
	}

	// "This party's over, so GTFO." -- Slaughter
	return queue.Success()
}
