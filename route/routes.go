package route

import (
	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

// New returns all of the routes required for this application
func New(factoryManager *server.FactoryManager) *echo.Echo {

	e := echo.New()

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.Static("/htmx", "../htmx/src")

	e.GET("/favicon.ico", echo.NotFoundHandler)
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factoryManager))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factoryManager))

	// RSS Feed
	e.GET("/feed.json", handler.GetRSS(factoryManager))

	e.Static("/r", "static")

	// Authentication Pages
	e.GET("/signin", handler.GetSignIn(factoryManager))
	e.POST("/signin", handler.PostSignIn(factoryManager))
	e.POST("/signout", handler.PostSignOut(factoryManager))

	// ActivityPub INBOX/OUTBOX
	e.GET("/users/:username/inbox", handler.TBD)
	e.POST("/users/:username/inbox", handler.TBD)
	e.GET("/users/:username/outbox", handler.TBD)
	e.POST("/users/:username/outbox", handler.TBD)

	// Stream Pages
	e.GET("/", handler.GetStream(factoryManager))                  // ?view=
	e.GET("/:stream", handler.GetStream(factoryManager))           // ?view= or ?transition=
	e.POST("/:stream", handler.PostStream(factoryManager))         // ?transition
	e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager)) // ?view=
	e.GET("/:stream/new", handler.GetNewTemplates(factoryManager))
	e.GET("/:stream/new/:template", handler.GetNewStream(factoryManager))
	e.POST("/:stream/new/:template", handler.PostNewStream(factoryManager))

	return e
}
