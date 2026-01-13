package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/collections"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/sherlock"
	"github.com/benpate/steranko"
)

// GetAPIActors returns a list of actors that match the provided search criteria.
// This is used in the E2EE service, as well as other actor lookups
func GetAPIActors(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetAPIActors"

	activityService := factory.ActivityStream()

	searchString := ctx.QueryParam("q")
	actors, err := activityService.QueryActors(searchString)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query actors", searchString)
	}

	return ctx.JSON(http.StatusOK, actors)
}

// GetAPICollectionHeader returns the header information for an ActivityPub collection
// (including totalItems and first page URL)
func GetAPICollectionHeader(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	const location = "handler.GetAPICollectionHeader"

	// Retrieve the collection from the network/cache
	url := ctx.QueryParam("url")
	activityService := factory.ActivityStream()
	document, err := activityService.AppClient().Load(url, sherlock.AsCollection())

	if err != nil {
		return derp.Wrap(err, location, "Unable to load collection", "url: "+url)
	}

	// Create a "header" object
	result := streams.CollectionHeader{
		ID:   document.ID(),
		Type: document.Type(),
	}

	if firstPage := calcCollectionFirstPage(document); firstPage != "" {
		result.First = firstPage
	}

	// Count the total items in the collection
	totalItems, err := collections.CountItems(document)

	if err != nil {
		return derp.Wrap(err, location, "Unable to count collection items", "url: "+url)
	}

	result.TotalItems = totalItems

	// Return the result as ActivityPub JSON
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}

func calcCollectionFirstPage(collection streams.Document) string {

	// If the collection publishes a "first page" then just use that
	if firstPage := collection.First().String(); firstPage != "" {
		return firstPage
	}

	// If there are >0 inline items, then the collection itself is the first page
	if items := collection.Items(); items.NotNil() {
		if items.Len() > 0 {
			return collection.ID()
		}
	}

	// Otherwise, there aren't any items, so there's no first page
	return ""
}
