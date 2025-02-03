package consumer

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
)

func IndexAllUsers(factory *domain.Factory, args mapof.Any) queue.Result {

	const location = "consumer.IndexAllUsers"

	searchService := factory.SearchResult()
	userService := factory.User()

	allUsers, err := userService.RangeAll()

	if err != nil {
		return queue.Error(derp.Wrap(err, location, "Error retrieving Users"))
	}

	for user := range allUsers {

		searchResult := userService.SearchResult(&user)

		if err := searchService.Sync(searchResult); err != nil {
			derp.Report(derp.Wrap(err, location, "Error saving SearchResult"))
		}
	}

	return queue.Success()
}
