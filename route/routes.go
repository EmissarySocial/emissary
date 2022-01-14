package route

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/middleware"
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
	e.Static("/static", "templates/system/static")

	// RSS Feed
	e.GET("/feed.json", handler.GetRSS(factoryManager))

	// Authentication Pages
	e.GET("/signin", handler.GetSignIn(factoryManager))
	e.POST("/signin", handler.PostSignIn(factoryManager))
	e.POST("/signout", handler.PostSignOut(factoryManager))

	// STREAM PAGES
	e.GET("/", handler.GetStream(factoryManager))
	e.GET("/:stream", handler.GetStream(factoryManager))
	e.GET("/:stream/:action", handler.GetStream(factoryManager))
	e.POST("/:stream/:action", handler.PostStream(factoryManager))
	e.DELETE("/:stream", handler.PostStream(factoryManager))

	// TODO: Can Attachments and SSE be moved into a custom render step?

	// SERVER ADMIN PAGES
	serverAdmin := e.Group("", middleware.ServerAdmin(factoryManager))
	serverAdmin.GET("/server", handler.GetServerIndex(factoryManager))
	serverAdmin.GET("/server/:domain", handler.GetServerDomain(factoryManager))
	serverAdmin.POST("/server/:domain", handler.PostServerDomain(factoryManager))
	serverAdmin.DELETE("/server/:domain", handler.DeleteServerDomain(factoryManager))

	// DOMAIN ADMIN PAGES
	e.GET("/admin", handler.GetAdmin(factoryManager))
	e.GET("/admin/:param1", handler.GetAdmin(factoryManager))
	e.POST("/admin/:param1", handler.PostAdmin(factoryManager))
	e.GET("/admin/:param1/:param2", handler.GetAdmin(factoryManager))
	e.POST("/admin/:param1/:param2", handler.PostAdmin(factoryManager))
	e.GET("/admin/:param1/:param2/:param3", handler.GetAdmin(factoryManager))
	e.POST("/admin/:param1/:param2/:param3", handler.PostAdmin(factoryManager))

	// Hard-coded routes for additional stream services
	e.GET("/:stream/attachments/:attachment", handler.GetAttachment(factoryManager))
	e.GET("/:stream/sse", handler.ServerSentEvent(factoryManager))

	// ActivityPub INBOX/OUTBOX
	e.GET("/inbox", handler.GetInbox(factoryManager))
	e.POST("/inbox", handler.PostInbox(factoryManager))
	e.GET("/outbox", handler.GetOutbox(factoryManager))
	e.POST("/outbox", handler.PostOutbox(factoryManager))

	// PROFILE PAGES
	// e.GET("/me/", handler.GetProfile(factoryManager))
	// e.POST("/me", handler.PostProfile(factoryManager))
	// e.GET("/me/:action", handler.PostProfile(factoryManager))
	// e.POST("/me/:action", handler.PostProfile(factoryManager))

	// SITE-WIDE ERROR HANDLER
	e.HTTPErrorHandler = func(err error, ctx echo.Context) {

		// If Forbidden error, then redirect the user to the signin page.
		if derp.ErrorCode(err) == derp.CodeForbiddenError {
			ctx.Redirect(http.StatusTemporaryRedirect, "/signin")
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
