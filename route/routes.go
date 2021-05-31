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
	e.Use(middleware.Steranko(factoryManager))

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.Static("/htmx", "../htmx/src")
	e.Static("/hyperscript", "../_hyperscript/src/lib")

	e.GET("/favicon.ico", echo.NotFoundHandler)
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factoryManager))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factoryManager))
	e.GET("/.well-known/oembed", handler.GetOEmbed(factoryManager))

	// RSS Feed
	e.GET("/feed.json", handler.GetRSS(factoryManager))

	e.Static("/static", "templates/static")

	// Authentication Pages
	e.GET("/signin", handler.GetSignIn(factoryManager))
	e.POST("/signin", handler.PostSignIn(factoryManager))
	e.POST("/signout", handler.PostSignOut(factoryManager))

	// ActivityPub INBOX/OUTBOX
	e.GET("/inbox", handler.GetInbox(factoryManager))
	e.POST("/inbox", handler.PostInbox(factoryManager))
	e.GET("/outbox", handler.GetOutbox(factoryManager))
	e.POST("/outbox", handler.PostOutbox(factoryManager))

	/*
		// Stream Pages
		e.GET("/", handler.GetStream(factoryManager))        // ?view=
		e.GET("/:stream", handler.GetStream(factoryManager)) // ?view=
		e.GET("/:stream/draft", handler.GetStreamDraft(factoryManager))
		e.POST("/:stream/draft", handler.PostStreamDraft(factoryManager))
		e.DELETE("/:stream/draft", handler.DeleteStreamDraft(factoryManager))
		e.POST("/:stream/draft/publish", handler.PublishStreamDraft(factoryManager))
		e.GET("/:stream/transition/:transition", handler.GetTransition(factoryManager))
		e.POST("/:stream/transition/:transition", handler.PostTransition(factoryManager)) // ?transition
		e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager))                    // ?view=
		e.GET("/:stream/new", handler.GetTemplates(factoryManager))
		e.GET("/:stream/new/:template", handler.GetNewStreamFromTemplate(factoryManager))
		e.POST("/:stream/new/:template", handler.PostNewStreamFromTemplate(factoryManager))
		e.GET("/:stream/layout/:file", handler.GetLayout(factoryManager))
	*/

	/// REFACTORED STREAM PAGES

	e.GET("/", handler.GetStream(factoryManager))
	e.GET("/:stream", handler.GetStream(factoryManager))
	e.GET("/:stream/:action", handler.GetAction(factoryManager))
	e.POST("/:stream/:action", handler.PostAction(factoryManager))
	e.DELETE("/:stream", handler.PostAction(factoryManager))
	e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager))
	e.GET("/:stream/layout/:file", handler.GetLayout(factoryManager))

	return e
}
