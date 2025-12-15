package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

// GetEmptyCollection returns an empty collection
func GetEmptyCollection(ctx *steranko.Context, factory *service.Factory, _ data.Session) error {

	// Create empty collection JSON-LD
	result := mapof.Any{
		vocab.AtContext:          vocab.ContextTypeActivityStreams,
		vocab.PropertyType:       vocab.CoreTypeOrderedCollection,
		vocab.PropertyID:         fullURL(factory, ctx),
		vocab.PropertyTotalItems: 0,
		vocab.PropertyItems:      []any{},
	}

	// Return success
	ctx.Response().Header().Set("Content-Type", vocab.ContentTypeActivityPub)
	return ctx.JSON(http.StatusOK, result)
}
