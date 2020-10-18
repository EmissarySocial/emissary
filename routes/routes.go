package routes

import (
	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/middleware"
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
	e.GET("/:stream", handler.GetStream(factoryManager)) // query param ?view=
	e.GET("/:stream/html", handler.GetStream(factoryManager), middleware.MimeType("text/html"))
	e.GET("/:stream/json", handler.GetStream(factoryManager), middleware.MimeType("application/json"))
	e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager))          // query param ?view=
	e.GET("/:stream/new/:template", handler.GetNewStream(factoryManager))
	e.POST("/:stream/new/:template", handler.PostNewStream(factoryManager))
	e.GET("/:stream/form/:transitionId", handler.GetForm(factoryManager))   // view a form (partial)
	e.POST("/:stream/form/:transitionId", handler.PostForm(factoryManager)) // post a form (with redirect)

	return e
}
