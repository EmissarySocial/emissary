package routes

import (
	"context"

	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/middleware"
	"github.com/benpate/ghost/service"
	"github.com/benpate/presto"
	"github.com/benpate/presto/scope"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

// New returns all of the routes required for this application
func New(factoryMaker service.FactoryMaker) *echo.Echo {

	domainWrapper := middleware.DomainWrapper()

	// BACKGROUND TASKS HERE (probably move to another file...)

	backgroundFactory := factoryMaker.Factory(context.Background())

	// Listen for updates to Streams
	streamChannel := service.Watcher(viper.GetString("dbserver"), viper.GetString("dbname"))
	broker := handler.NewBroker(backgroundFactory, streamChannel)

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
	e.GET("/:token", handler.GetStream(factoryMaker), domainWrapper)
	e.GET("/:token/", handler.GetStream(factoryMaker), domainWrapper)
	e.GET("/:token/:view", handler.GetStream(factoryMaker), domainWrapper)
	e.GET("/:token/:view/sse", handler.ServerSentEvent(broker))
	e.GET("/:token/:view/websocket", handler.Websocket(broker))

	e.Static("/htmx", "/Users/benpate/Documents/Source Code/github.com/benpate/htmx/src")

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
