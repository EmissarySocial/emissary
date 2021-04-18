package route

import (
	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/middleware"
	"github.com/benpate/ghost/server"
	"github.com/labstack/echo/v4"
)

// New returns all of the routes required for this application
func New(factoryManager *server.FactoryManager) *echo.Echo {

	e := echo.New()

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.Static("/htmx", "../htmx/src")
	e.Static("/hyperscript", "../_hyperscript/src/lib")

	e.GET("/favicon.ico", echo.NotFoundHandler)
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factoryManager))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factoryManager))

	// RSS Feed
	e.GET("/feed.json", handler.GetRSS(factoryManager))

	e.Static("/static", "templates/static")

	// Authentication Pages
	e.GET("/signin", handler.GetSignIn(factoryManager))
	e.POST("/signin", handler.PostSignIn(factoryManager))
	e.POST("/signout", handler.PostSignOut(factoryManager))

	// ActivityPub INBOX/OUTBOX
	e.GET("/users/:username/inbox", handler.TBD, middleware.TrySignin)
	e.POST("/users/:username/inbox", handler.TBD, middleware.TrySignin)
	e.GET("/users/:username/outbox", handler.TBD, middleware.TrySignin)
	e.POST("/users/:username/outbox", handler.TBD, middleware.TrySignin)

	// Stream Pages
	e.GET("/", handler.GetStream(factoryManager), middleware.TrySignin)        // ?view=
	e.GET("/:stream", handler.GetStream(factoryManager), middleware.TrySignin) // ?view= or ?transition=
	e.GET("/:stream/transition/:transition", handler.GetTransition(factoryManager), middleware.TrySignin)
	e.POST("/:stream/transition/:transition", handler.PostTransition(factoryManager), middleware.TrySignin) // ?transition
	e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager), middleware.TrySignin)                    // ?view=
	e.GET("/:stream/new", handler.GetTemplates(factoryManager), middleware.TrySignin)
	e.GET("/:stream/new/:template", handler.GetNewStreamFromTemplate(factoryManager), middleware.TrySignin)
	e.POST("/:stream/new/:template", handler.PostNewStreamFromTemplate(factoryManager), middleware.TrySignin)
	e.GET("/:stream/layout/:file", handler.GetLayout(factoryManager), middleware.TrySignin)

	return e
}
