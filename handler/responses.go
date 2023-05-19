package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/labstack/echo/v4"
)

func GetLikes(serverFactory *server.Factory) echo.HandlerFunc {
	return handleResponseCollection(serverFactory, model.ResponseTypeLike)
}

func GetDislikes(serverFactory *server.Factory) echo.HandlerFunc {
	return handleResponseCollection(serverFactory, model.ResponseTypeDislike)
}

func GetMentions(serverFactory *server.Factory) echo.HandlerFunc {
	return handleResponseCollection(serverFactory, model.ResponseTypeMention)
}

func handleResponseCollection(serverFactory *server.Factory, responseType string) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		const location = "handler.handleResponseCollection"

		factory, err := serverFactory.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error creating factory")
		}

		streamService := factory.Stream()
		stream := model.NewStream()
		token := ctx.Param("stream")

		if err := streamService.LoadByToken(token, &stream); err != nil {
			return derp.Wrap(err, location, "Error loading stream")
		}

		// RULE: Only PUBLIC streams have /likes /dislikes and /mentions
		if !stream.DefaultAllowAnonymous() {
			return derp.NewUnauthorizedError(location, "Anonymous access not allowed")
		}

		// Query all responses for this Stream
		streamResponseService := factory.StreamResponse()
		responses, err := streamResponseService.QueryByStreamAndType(stream.StreamID, responseType)

		if err != nil {
			return derp.Wrap(err, location, "Error loading responses")
		}

		// Return a JSON-LD document
		ctx.Response().Header().Set("Content-Type", model.MimeTypeActivityPub)

		result := streams.Collection{
			Context:    streams.DefaultContext(),
			Type:       vocab.CoreTypeCollection,
			TotalItems: stream.Responses.CountByType(responseType),
			Items: slice.Map(responses, func(item model.StreamResponse) any {
				return mapof.Any{
					"id":    item.Origin.URL,
					"actor": item.Actor.ProfileURL,
				}
			}),
		}

		return ctx.JSON(http.StatusOK, result)
	}
}
