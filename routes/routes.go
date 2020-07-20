package routes

import (
	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/service"
	"github.com/benpate/presto"
	"github.com/benpate/presto/scope"
	"github.com/labstack/echo/v4"
)

// New returns all of the routes required for this application
func New(factoryMaker service.FactoryMaker) *echo.Echo {

	e := echo.New()

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.GET("/.well-known/webfinger", handler.GetWebfinger(factoryMaker))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factoryMaker))

	// RSS Feed
	e.GET("/feed.json", handler.GetRSS(factoryMaker))

	// Home Page for the website (should probably be a redirect to a "default" space?)
	e.GET("/", handler.TBD)

	// Stream Pages
	e.GET("/:stream", handler.GetStream(factoryMaker))
	e.GET("/:stream/", handler.GetStream(factoryMaker))

	e.GET("/:stream/:view", handler.GetStream(factoryMaker))

	// Presto Global Settings
	presto.UseRouter(e)
	presto.UseScopes(scope.NotDeleted)

	presto.NewCollection(factoryMaker.Stream, "/streams").
		UseScopes().
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factoryMaker.Attachment, "/streams/:stream/pages/:page/attachments").
		UseScopes(scope.String("stream", "page")).
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factoryMaker.Comment, "/streams/:stream/pages/:page/comments").
		UseScopes(scope.String("stream", "page")).
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factoryMaker.User, "/users/:username").
		UseScopes().
		List().
		Post().
		Get().
		Put().
		Delete()

	// ActivityPub INBOX/OUTBOX
	e.GET("/users/:username/inbox", handler.TBD)
	e.POST("/users/:username/inbox", handler.TBD)
	e.GET("/users/:username/outbox", handler.TBD)
	e.POST("/users/:username/outbox", handler.TBD)

	return e
}
