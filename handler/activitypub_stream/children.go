package activitypub_stream

import (
	"net/http"

	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetChildrenCollection returns a collection of all child streams for a given parent stream
func GetChildrenCollection(ctx *steranko.Context, factory *service.Factory, session data.Session, parent *model.Stream) error {

	const location = "handler.activitypub_stream.GetChildrenCollection"

	// Calculate the caller's permissions from the HTTP signature (defaults to
	// anonymous when the request is unsigned).
	permissions := factory.Permission().ParseHTTPSignature(session, ctx.Request())

	// Get an iterator of the currently-published child streams. The publish-date
	// window is applied in the query; per-child permissions are applied below.
	streamService := factory.Stream()
	children, err := streamService.RangePublishedByParent(session, parent.StreamID)

	if err != nil {
		return derp.Wrap(err, location, "Loading children")
	}

	// Map each VISIBLE child into JSON and stuff it into the collection's OrderedItems.
	result := activitypub.Collection(parent.ActivityPubChildrenURL())
	count := 0
	for child := range children {

		// RULE: Skip children the caller is not permitted to view.
		if !child.IsVisibleTo(permissions) {
			continue
		}

		childJSON := streamService.JSONLD(session, &child)
		result.OrderedItems = append(result.OrderedItems, childJSON)
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
