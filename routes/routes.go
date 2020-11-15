package routes

import (
	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/service"
	"github.com/labstack/echo/v4"
)

// New returns all of the routes required for this application
func New(factoryManager *service.FactoryManager) *echo.Echo {

	e := echo.New()

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.GET("/favicon.ico", echo.NotFoundHandler)
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factoryManager))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factoryManager))

	// RSS Feed
	e.GET("/feed.json", handler.GetRSS(factoryManager))

	e.Static("/r", "static")

	// ActivityPub INBOX/OUTBOX
	e.GET("/users/:username/inbox", handler.TBD)
	e.POST("/users/:username/inbox", handler.TBD)
	e.GET("/users/:username/outbox", handler.TBD)
	e.POST("/users/:username/outbox", handler.TBD)

	// Stream Pages
	e.GET("/", handler.GetStream(factoryManager)) // query param ?view=

	e.GET("/:stream", handler.GetStream(factoryManager))           // query param ?view=
	e.POST("/:stream", handler.PostStream(factoryManager))         // post a form (with redirect)
	e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager)) // query param ?view=
	e.GET("/new", handler.GetNewStream(factoryManager))
	e.POST("/new", handler.PostNewStream(factoryManager))

	return e
}
