package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

func IndexAllUsers(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.IndexAllUsers"

	searchService := factory.Search()
	userService := factory.User()

	allUsers, err := userService.RangeAll()

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error retrieving Users"))
	}

	for user := range allUsers {

		searchResult, _ := userService.SearchResult(&user)

		// If user is indexable, then add/update them in the search index
		if user.IsIndexable {

			if err := searchService.Upsert(searchResult); err != nil {
				derp.Report(derp.Wrap(err, location, "Error saving SearchResult"))
			}

			continue
		}

		// If NOT indexable, then remove them from the search index
		if err := searchService.DeleteByURL(searchResult.URL); err != nil {
			derp.Report(derp.Wrap(err, location, "Error deleting SearchResult"))
		}
	}

	return queue.Success()
}
