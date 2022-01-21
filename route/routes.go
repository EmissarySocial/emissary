package route

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"github.com/whisperverse/whisperverse/handler"
	"github.com/whisperverse/whisperverse/middleware"
	"github.com/whisperverse/whisperverse/server"
)

// New returns all of the routes required for this application
func New(factory *server.Factory) *echo.Echo {

	e := echo.New()

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.GET("/favicon.ico", handler.GetFavicon(factory))
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factory))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factory))
	e.GET("/.well-known/oembed", handler.GetOEmbed(factory))

	// Local links for static resources
	e.Static("/static", factory.StaticPath())

	// RSS Feed
	e.GET("/feed.json", handler.GetRSS(factory))

	// Authentication Pages
	e.GET("/signin", handler.GetSignIn(factory))
	e.POST("/signin", handler.PostSignIn(factory))
	e.POST("/signout", handler.PostSignOut(factory))

	// STREAM PAGES
	e.GET("/", handler.GetStream(factory))
	e.GET("/:stream", handler.GetStream(factory))
	e.GET("/:stream/:action", handler.GetStream(factory))
	e.POST("/:stream/:action", handler.PostStream(factory))
	e.DELETE("/:stream", handler.PostStream(factory))

	// TODO: Can Attachments and SSE be moved into a custom render step?

	// SERVER ADMIN PAGES
	serverAdmin := middleware.ServerAdmin(factory)
	e.GET("/server", handler.GetServerIndex(factory), serverAdmin)
	e.GET("/server/:domain", handler.GetServerDomain(factory), serverAdmin)
	e.POST("/server/:domain", handler.PostServerDomain(factory), serverAdmin)
	e.DELETE("/server/:domain", handler.DeleteServerDomain(factory), serverAdmin)

	// DOMAIN ADMIN PAGES
	e.GET("/admin", handler.GetAdmin(factory))
	e.GET("/admin/:param1", handler.GetAdmin(factory))
	e.POST("/admin/:param1", handler.PostAdmin(factory))
	e.GET("/admin/:param1/:param2", handler.GetAdmin(factory))
	e.POST("/admin/:param1/:param2", handler.PostAdmin(factory))
	e.GET("/admin/:param1/:param2/:param3", handler.GetAdmin(factory))
	e.POST("/admin/:param1/:param2/:param3", handler.PostAdmin(factory))

	// Hard-coded routes for additional stream services
	e.GET("/:stream/attachments/:attachment", handler.GetAttachment(factory))
	e.GET("/:stream/sse", handler.ServerSentEvent(factory))

	// ActivityPub INBOX/OUTBOX
	e.GET("/inbox", handler.GetInbox(factory))
	e.POST("/inbox", handler.PostInbox(factory))
	e.GET("/outbox", handler.GetOutbox(factory))
	e.POST("/outbox", handler.PostOutbox(factory))

	// PROFILE PAGES
	// e.GET("/me/", handler.GetProfile(factory))
	// e.POST("/me", handler.PostProfile(factory))
	// e.GET("/me/:action", handler.PostProfile(factory))
	// e.POST("/me/:action", handler.PostProfile(factory))

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
