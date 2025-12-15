package activitypub_search

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/steranko"
)

func GetJSONLD(ctx *steranko.Context, factory *service.Factory, session data.Session, template *model.Template, stream *model.Stream, searchQuery *model.SearchQuery) error {

	const location = "handler.activitypub_search.GetJSONLD"

	// Retrieve the JSON-LD for this SearchQuery
	searchQueryService := factory.SearchQuery()
	result, err := searchQueryService.GetJSONLD(session, searchQuery)

	if err != nil {
		return derp.Wrap(err, location, "Unable to generate JSON-LD for search query actor")
	}

	// Return the JSON-LD to the caller
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(200, result)
}
