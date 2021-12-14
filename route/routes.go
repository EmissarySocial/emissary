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

	e.GET("/favicon.ico", handler.GetFavicon(factoryManager))
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factoryManager))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factoryManager))
	e.GET("/.well-known/oembed", handler.GetOEmbed(factoryManager))

	// Local links for static resources
	e.Static("/htmx", "../htmx/src")
	e.Static("/hyperscript", "../_hyperscript/dist")
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

	// DOMAIN ADMIN
	e.GET("/domain", handler.GetDomain(factoryManager))
	e.GET("/domain/:action", handler.GetDomain(factoryManager))
	e.POST("/domain/:action", handler.PostDomain(factoryManager))

	// STREAM PAGES
	e.GET("/", handler.GetStream(factoryManager))
	e.GET("/:stream", handler.GetStream(factoryManager))
	e.GET("/:stream/:action", handler.GetStream(factoryManager))
	e.POST("/:stream/:action", handler.PostStream(factoryManager))
	e.DELETE("/:stream", handler.PostStream(factoryManager))

	// TODO: Can Attachments and SSE be moved into a custom render step?
	e.GET("/:stream/attachments/:attachment", handler.GetAttachment(factoryManager))
	e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager))

	// SITE-WIDE ERROR HANDLER
	e.HTTPErrorHandler = func(err error, ctx echo.Context) {

		// If Forbidden error, then redirect the user to the signin page.
		if derp.ErrorCode(err) == derp.CodeForbiddenError {
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
