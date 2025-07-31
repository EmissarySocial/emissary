package activitypub_stream

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
)

// fullURL returns the URL for a request that include the protocol, hostname, and path
func fullURL(factory *domain.Factory, ctx echo.Context) string {
	return factory.Host() + ctx.Request().URL.String()
}

func getResponseType(ctx *steranko.Context) string {

	switch list.Last(ctx.Request().URL.Path, '/') {

	case "shared":
		return vocab.ActivityTypeAnnounce

	case "liked":
		return vocab.ActivityTypeLike

	case "disliked":
		return vocab.ActivityTypeDislike
	}

	return ""
}
