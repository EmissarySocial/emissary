package route

import (
	"net/http"
	"net/url"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/server"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
)

// New returns all of the routes required for this application
func New(factoryManager *server.FactoryManager) *echo.Echo {

	e := echo.New()

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.GET("/favicon.ico", echo.NotFoundHandler)
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factoryManager))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factoryManager))
	e.GET("/.well-known/oembed", handler.GetOEmbed(factoryManager))

	// Local links for static resources
	e.Static("/htmx", "../htmx/src")
	e.Static("/hyperscript", "../_hyperscript/src/lib")
	e.Static("/static", "templates/static")

	// RSS Feed
	e.GET("/feed.json", handler.GetRSS(factoryManager))

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
	e.GET("/:stream/:action", handler.GetStream(factoryManager))
	e.POST("/:stream/:action", handler.PostStream(factoryManager))
	e.DELETE("/:stream", handler.PostStream(factoryManager))
	e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager))
	e.GET("/:stream/layout/:file", handler.GetLayout(factoryManager))

	// CUSTOM ERROR HANDLER

	e.HTTPErrorHandler = func(err error, ctx echo.Context) {

		errorCode := derp.ErrorCode(err)

		switch errorCode {
		case derp.CodeForbiddenError:
			ctx.Redirect(http.StatusTemporaryRedirect, "/signin?next="+url.QueryEscape(ctx.Request().RequestURI))
			return
		}

		// Fall through to general error handler
		if ctx.Request().Host == "localhost" {
			ctx.String(derp.ErrorCode(err), spew.Sdump(err))
		}
		ctx.String(derp.ErrorCode(err), derp.Message(err))
	}

	return e
}
