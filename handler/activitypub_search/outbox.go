package activitypub_search

import (
	"math"
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/handler/activitypub"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
)

func GetOutboxCollection(ctx *steranko.Context, factory *domain.Factory, session data.Session, template *model.Template, _ *model.Stream, searchQuery *model.SearchQuery) error {

	const location = "handler.activitypub_search.GetOutboxCollection"

	searchQueryService := factory.SearchQuery()
	actorID := searchQueryService.ActivityPubURL(searchQuery.SearchQueryID)
	outboxURL := searchQueryService.ActivityPubOutboxURL(searchQuery.SearchQueryID)

	// If the request is for the collection itself, then return a summary and the URL of the first page
	publishDateString := ctx.QueryParam("publishDate")

	if publishDateString == "" {
		ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
		result := activitypub.Collection(outboxURL)
		return ctx.JSON(http.StatusOK, result)
	}

	// Fall through means that we're looking for a specific page of the collection
	publishedDate := convert.Int64Default(publishDateString, math.MaxInt64)
	pageID := fullURL(factory, ctx)
	pageSize := 60

	// Retrieve a page of messages from the database

	criteria := searchQuery.Expression()
	criteria = criteria.AndLessThan("createDate", publishedDate)

	searchResultService := factory.SearchResult()
	results, err := searchResultService.Query(session, criteria, option.SortDesc("createDate"), option.MaxRows(60))

	if err != nil {
		return derp.Wrap(err, location, "Error retrieving search results")
	}

	activities := slice.Map(results, mapSearchResult(actorID))

	// Return results as an OrderedCollectionPage
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	result := activitypub.CollectionPage(pageID, outboxURL, pageSize, activities)
	return ctx.JSON(http.StatusOK, result)
}

func mapSearchResult(actorID string) func(r model.SearchResult) model.JSONLD {

	return func(r model.SearchResult) model.JSONLD {

		return model.JSONLD{
			vocab.AtContext:         vocab.ContextTypeActivityStreams,
			vocab.PropertyID:        actorID + "/pub/outbox/" + r.SearchResultID.Hex(),
			vocab.PropertyActor:     actorID,
			vocab.PropertyType:      vocab.ActivityTypeAnnounce,
			vocab.PropertyObject:    r.URL,
			vocab.PropertyPublished: r.CreateDate,
		}
	}
}
