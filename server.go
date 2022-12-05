package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/handler"
	mw "github.com/EmissarySocial/emissary/middleware"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/tools/domain"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/browser"
)

//go:embed all:_embed/**
var embeddedFiles embed.FS

func main() {

	fmt.Println("Starting Emissary.")

	// Set global configuration
	spew.Config.DisableMethods = true
	spew.Config.Indent = " "

	// Locate the configuration file and populate the factory
	commandLineArgs := config.GetCommandLineArgs()
	configStorage := config.Load(commandLineArgs)

	factory := server.NewFactory(configStorage, embeddedFiles)

	// Start and configure the Web server
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = errorHandler

	// Global middleware
	e.Use(middleware.Recover())
	// TODO: HIGH: implement echo.Security middleware

	// Static Content MUST come from static files, because we're not re-linking all those files at runtime
	if staticFiles, err := fs.Sub(embeddedFiles, "_embed"); err == nil {

		e.Group("/static", mw.CacheControl("public, max-age=3600")).
			GET("/*", echo.WrapHandler(http.FileServer(http.FS(staticFiles))))

	} else {
		panic(err)
	}

	// Based on configuration, add all other routes and start web server
	if commandLineArgs.Setup {
		makeSetupRoutes(factory, e)
	} else {
		makeStandardRoutes(factory, e)
	}
}

// makeSetupRoutes generates a new Echo instance for the setup behavior
func makeSetupRoutes(factory *server.Factory, e *echo.Echo) {

	fmt.Println("Starting Emissary Config Tool.")

	// Locate the setup templates
	setupFiles, err := fs.Sub(embeddedFiles, "_embed/setup")

	if err != nil {
		panic("Unable to open embedded files for setup. " + err.Error())
	}

	setupTemplates := template.Must(template.New("").
		Funcs(factory.FuncMap()).
		ParseFS(setupFiles, "*.html"))

	// Middleware for setup pages
	// TODO: LOW: Security
	// TODO: LOW: Rate Limiter
	// TODO: HIGH: CSRF
	e.Use(mw.Localhost())

	// Setup Routes
	e.GET("/", handler.SetupPageGet(factory, setupTemplates, "index.html"))
	e.GET("/server", handler.SetupPageGet(factory, setupTemplates, "server.html"))
	e.POST("/server", handler.SetupServerPost(factory))
	e.GET("/server/:section", handler.SetupServerGet(factory))
	e.POST("/server/:section", handler.SetupServerPost(factory))
	e.GET("/domains", handler.SetupPageGet(factory, setupTemplates, "domains.html"))
	e.GET("/domains/:domain", handler.SetupDomainGet(factory))
	e.POST("/domains/:domain", handler.SetupDomainPost(factory))
	e.DELETE("/domains/:domain", handler.SetupDomainDelete(factory))
	e.POST("/domains/:domain/signin", handler.SetupDomainSigninPost(factory))
	e.GET("/domains/:domain/users", handler.SetupDomainUsersGet(factory, setupTemplates))
	e.POST("/domains/:domain/users", handler.SetupDomainUserPost(factory, setupTemplates))
	e.POST("/domains/:domain/users/:user/invite", handler.SetupDomainUserInvite(factory, setupTemplates))
	e.DELETE("/domains/:domain/users/:user", handler.SetupDomainUserDelete(factory, setupTemplates))
	e.GET("/oauth", handler.SetupOAuthList(factory, setupTemplates))
	e.GET("/oauth/:provider", handler.SetupOAuthGet(factory, setupTemplates))
	e.POST("/oauth/:provider", handler.SetupOAuthPost(factory, setupTemplates))

	// When running the setup tool, wait a second, then open a browser window to the correct URL
	go func() {
		time.Sleep(time.Second * 1)
		browser.OpenURL("http://localhost:8080/")
	}()

	// Start the HTTP server on alternate port 8080
	fmt.Println("Starting HTTP server...")
	if err := e.Start(":8080"); err != nil {
		derp.Report(derp.Wrap(err, "setup.Setup", "Error starting HTTP server"))
	}
}

// makeStandardRoutes generates a new Echo instance the primary server behavior
func makeStandardRoutes(factory *server.Factory, e *echo.Echo) {

	e.Pre(mw.HttpsRedirect)

	// Middleware for standard pages
	// TODO: MEDIUM: Rate Limiter
	// TODO: MEDIUM: Security Middleware
	e.Use(mw.Domain(factory))
	e.Use(steranko.Middleware(factory))

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.GET("/favicon.ico", handler.GetFavicon(factory))
	e.GET("/.well-known/change-password", handler.TBD)
	e.GET("/.well-known/host-meta", handler.TBD)
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factory))
	e.GET("/.well-known/oembed", handler.GetOEmbed(factory))
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factory))

	// Built-In Service  Routes
	e.POST("/.webmention", handler.PostWebMention(factory))
	e.GET("/.giphy", handler.GetGiphyWidget(factory))

	// Authentication Pages
	e.GET("/signin", handler.GetSignIn(factory))
	e.POST("/signin", handler.PostSignIn(factory))
	e.POST("/signout", handler.PostSignOut(factory))
	e.GET("/register", handler.GetRegister(factory))
	e.POST("/register", handler.PostRegister(factory))
	e.GET("/signin/reset", handler.GetResetPassword(factory))
	e.POST("/signin/reset", handler.PostResetPassword(factory))
	e.GET("/signin/reset-code", handler.GetResetCode(factory))
	e.POST("/signin/reset-code", handler.PostResetCode(factory))

	// STREAM PAGES
	e.GET("/", handler.GetStream(factory))
	e.GET("/:stream", handler.GetStream(factory))
	e.GET("/:stream/:action", handler.GetStream(factory))
	e.POST("/:stream/:action", handler.PostStream(factory))
	e.DELETE("/:stream", handler.PostStream(factory))

	// Hard-coded routes for additional stream services
	// TODO: LOW: Can Stream Attachments and SSE be moved into a custom render step?
	e.GET("/:stream/attachments/:attachment", handler.GetAttachment(factory))
	e.GET("/:stream/sse", handler.ServerSentEvent(factory))
	e.GET("/:stream/qrcode", handler.GetQRCode(factory))

	// Profile Pages
	// NOTE: these are rewritten from /@:userId by the rewrite middleware
	e.GET("/@", handler.TBD)
	e.GET("/@:userId", handler.GetProfile(factory))
	e.POST("/@:userId", handler.PostProfile(factory))
	e.GET("/@:userId/:action", handler.GetProfile(factory))
	e.POST("/@:userId/:action", handler.PostProfile(factory))
	e.GET("/@:userId/avatar", handler.GetProfileAvatar(factory))

	// ActivityPub Routes
	e.GET("/@:userId/pub/inbox", handler.ActivityPub_GetInbox(factory))
	e.GET("/@:userId/pub/inbox/:item", handler.ActivityPub_GenericHandler(factory))
	e.POST("/@:userId/pub/inbox", handler.ActivityPub_PostInbox(factory))

	e.GET("/@:userId/pub/outbox", handler.ActivityPub_GetOutbox(factory))
	e.GET("/@:userId/pub/outbox/:item", handler.ActivityPub_GenericHandler(factory))
	e.POST("/@:userId/pub/outbox", handler.ActivityPub_PostOutbox(factory))
	e.GET("/@:userId/pub/key", handler.ActivityPub_GetPublicKey(factory))
	e.GET("/@:userId/pub/followers", handler.ActivityPub_GenericHandler(factory))
	e.GET("/@:userId/pub/following", handler.ActivityPub_GenericHandler(factory))
	e.GET("/@:userId/pub/likes", handler.ActivityPub_GenericHandler(factory))

	// ME-ONLY PAGES
	e.POST("/@me/pub/inbox/:item/mark-read", handler.Activity_MarkRead(factory))
	e.POST("/@me/pub/inbox/:item/mark-unread", handler.Activity_MarkUnRead(factory))
	e.GET("/@me/pub/following/:following", handler.GetFollowing(factory))
	e.POST("/@me/pub/following/:following", handler.PostFollowing(factory))
	e.GET("/@me/pub/following/:following/delete", handler.GetDeleteFollowing(factory))
	e.POST("/@me/pub/following/:following/delete", handler.PostDeleteFollowing(factory))

	// DOMAIN ADMIN PAGES
	e.GET("/admin", handler.GetAdmin(factory), mw.Owner)
	e.GET("/admin/:param1", handler.GetAdmin(factory), mw.Owner)
	e.POST("/admin/:param1", handler.PostAdmin(factory), mw.Owner)
	e.GET("/admin/:param1/:param2", handler.GetAdmin(factory), mw.Owner)
	e.POST("/admin/:param1/:param2", handler.PostAdmin(factory), mw.Owner)
	e.GET("/admin/:param1/:param2/:param3", handler.GetAdmin(factory), mw.Owner)
	e.POST("/admin/:param1/:param2/:param3", handler.PostAdmin(factory), mw.Owner)

	// OAUTH Connections
	e.GET("/oauth/:provider", handler.GetOAuth(factory), mw.Owner)
	e.GET("/oauth/:provider/callback", handler.GetOAuthCallback(factory), mw.AllowCSR, mw.Owner)
	e.GET("/oauth/redirect", handler.OAuthRedirect(factory), mw.Owner)

	// Startup Wizard
	e.GET("/startup", handler.Startup(factory), mw.Owner)
	e.POST("/startup", handler.Startup(factory), mw.Owner)

	// EXTERNAL SERVICES (WEBHOOKS)
	e.POST("/webhooks/stripe", handler.StripeWebhook(factory))

	// Prepare HTTP and HTTPS servers using the new configuration
	go startHttps(e)
	go startHttp(e)

	// GRACEFUL SHUTDOWN FOR STANDARD SERVER
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

// startHttp starts the HTTPS server using Let's Encrypt SSL certificates.  If port 443 is not available, it will wait 10ms and retry until it is
func startHttps(e *echo.Echo) {
	fmt.Println("Starting HTTP server...")
	for {
		if err := e.StartAutoTLS(":443"); err != nil {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// startHttp starts the HTTP server.  If port 80 is not available, it will wait 10ms and retry until it is
func startHttp(e *echo.Echo) {
	fmt.Println("Starting HTTP server...")
	for {
		if err := e.Start(":80"); err != nil {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// errorHandler is a custom error handler that returns a JSON error message to the client
func errorHandler(err error, ctx echo.Context) {

	// Special handling of permisssion errors
	code := derp.ErrorCode(err)
	switch code {
	case http.StatusUnauthorized:

		if ctx.Request().URL.Path != "/signin" {
			ctx.Redirect(http.StatusTemporaryRedirect, "/signin")
			return
		}
		ctx.String(code, derp.Message(err))
		return
	}

	// On localhost, allow developers to see full error dump.
	if domain.IsLocalhost(ctx.Request().Host) {
		ctx.JSONPretty(derp.ErrorCode(err), err, "  ")
		return
	}

	// Fall through to general error handler
	ctx.JSONPretty(derp.ErrorCode(err), err, "  ")
	// ctx.String(derp.ErrorCode(err), derp.Message(err))
}

/** AUTOCERT HOST POLICY
TODO: Move this into Factory, or somewhere that can listen to configuration changes...

	// Find all NON-LOCAL domain names
	domains := slice.Filter(c.DomainNames(), isRemoteDomain)

	if len(domains) == 0 {
		fmt.Println("Skipping HTTPS server because there are no non-local domains.")
		return
	}

	fmt.Println("Starting HTTPS server...")

	// Initialize Let's Encrypt autocert for TLS certificates
	e.AutoTLSManager = autocert.Manager{
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache(c.Certificates.Location),
		Prompt:     autocert.AcceptTOS,
		Email:      c.AdminEmail,
	}



**/
