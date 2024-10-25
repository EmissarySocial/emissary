package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetChildrenCollection returns a collection of all child streams for a given parent stream
func GetChildrenCollection(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.activitypub_stream.GetChildrenCollection"

	streamService := factory.Stream()
	token := ctx.Param("stream")

	// Load the parent stream information
	parent := model.NewStream()
	if err := streamService.LoadByToken(token, &parent); err != nil {
		return derp.Wrap(err, location, "Error loading stream")
	}

	// Get an iterator of all child streams
	result := activitypub.Collection(parent.ActivityPubChildrenURL())
	children, err := streamService.ListByParent(parent.StreamID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading children")
	}

	// Map each child into JSON and stuff it into the collection's OrderedItems
	child := model.NewStream()
	count := 0
	for children.Next(&child) {
		childJSON := streamService.JSONLD(&child)
		result.OrderedItems = append(result.OrderedItems, childJSON)
		child = model.NewStream()
		count++
	}

	// Additional metadata
	result.TotalItems = count
	result.First = ""
	result.Summary = "Collection of all child streams for " + parent.Label

	// Return the result as JSON
	ctx.Response().Header().Set("Content-Type", "application/activity+json")
	return ctx.JSON(http.StatusOK, result)
}
