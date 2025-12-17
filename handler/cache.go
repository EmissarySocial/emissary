package handler

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/data"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReIndexActivityStreamCache is a handler function that queues up individual tasks to
// re-index each ActivityStream in the shared cache, one-by-one.
func ReIndexActivityStreamCache(ctx *steranko.Context, factory *service.Factory, session data.Session) error {

	activityStreamService := factory.ActivityStream(model.ActorTypeApplication, primitive.NilObjectID)
	iterator := activityStreamService.Range(context.Background(), exp.All())

	for cachedValue := range iterator {

		url := cachedValue.URLs.First()

		log.Debug().Str("url", url).Msg("Re-indexing ActivityStream")

		factory.Queue().NewTask(
			"ReindexActivityStream",
			mapof.Any{
				"host": factory.Hostname(),
				"url":  url,
			},
		)
	}

	return nil
}
