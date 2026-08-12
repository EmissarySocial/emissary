// Package activitypub_search serves the ActivityPub endpoints for per-SearchQuery actors: the
// JSON-LD profile of a saved search, and the outbox of results that search has matched.
//
// Each saved query is its own followable actor, distinct from the single domain-wide actor in
// activitypub_domain.
package activitypub_search

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/headers"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
)

// GetJSONLD generates JSON-LD for the actor that represents a saved SearchQuery
func GetJSONLD(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream, searchQuery *model.SearchQuery) error {

	const location = "handler.activitypub_search.GetJSONLD"

	// Retrieve the JSON-LD for this SearchQuery
	searchQueryService := factory.SearchQuery()
	result, err := searchQueryService.GetJSONLD(session, searchQuery)

	if err != nil {
		return derp.Wrap(err, location, "Generating JSON-LD for search query actor")
	}

	// Return the JSON-LD to the caller
	headers.SetVariant(ctx.Response().Header(), headers.VariantActivityPub)
	return ctx.JSON(200, result)
}
