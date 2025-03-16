/*
Package main is the entry point for the application.  It reads the server
configuration info,  initializes the server.Factory, wires up routes to
the appropriate handlers, then starts the HTTP/HTTPS server.
*/
package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/handler"
	ap_search "github.com/EmissarySocial/emissary/handler/activitypub_search"
	ap_stream "github.com/EmissarySocial/emissary/handler/activitypub_stream"
	ap_user "github.com/EmissarySocial/emissary/handler/activitypub_user"
	"github.com/EmissarySocial/emissary/handler/stripe"
	"github.com/EmissarySocial/emissary/handler/unsplash"
	mw "github.com/EmissarySocial/emissary/middleware"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome4echo"
	"github.com/benpate/domain"
	"github.com/benpate/form/widget"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	gommonlog "github.com/labstack/gommon/log"
	"github.com/pkg/browser"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/acme/autocert"
)

//go:embed all:_embed/**
var embeddedFiles embed.FS

/******************************************
 * Main Application Entry Point
 ******************************************/

func main() {

	fmt.Println(" _____           _                          ")
	fmt.Println("| ____|_ __ ___ (_)___ ___  __ _ _ __ _   _ ")
	fmt.Println("|  _| | '_ ` _ \\| / __/ __|/ _` | '__| | | |")
	fmt.Println("| |___| | | | | | \\__ \\__ \\ (_| | |  | |_| |")
	fmt.Println("|_____|_| |_| |_|_|___/___/\\__,_|_|   \\__, |")
	fmt.Println("                                      |___/ ")
	fmt.Println("")

	go waitForSigInt()

	// Troubleshoot / Error Reporting
	spew.Config.DisableMethods = true
	spew.Config.Indent = " "

	// Logging Configuration
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		NoColor:    true,
		TimeFormat: "",
	})

	// Configure form library
	widget.UseAll()

	// Locate the configuration file and populate the server factory
	commandLineArgs := config.GetCommandLineArgs()
	configStorage := config.Load(&commandLineArgs)

	factory := server.NewFactory(configStorage, embeddedFiles)

	// Start and configure the Web server
	e := echo.New()
	e.Logger.SetLevel(gommonlog.OFF)
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = errorHandler

	// Global middleware
	// TODO: HIGH: Implement echo.Secure - https://echo.labstack.com/docs/middleware/secure
	// TODO: HIGH: Implement CSRF protection - https://echo.labstack.com/docs/middleware/csrf
	// TODO: MEDIUM: Implement Rate Limiter - https://echo.labstack.com/docs/middleware/rate-limiter
	// TODO: LOW: Implement Timeout - https://echo.labstack.com/docs/middleware/timeout
	// TODO: LOW: Implement GZip - https://echo.labstack.com/docs/middleware/gzip
	e.Use(middleware.Recover())

	// Wait for the first time the configuration is loaded
	<-factory.Ready()

	if commandLineArgs.Setup {

		// Get config modifiers from the command line (like HTTP PORT)
		configOptions := commandLineArgs.ConfigOptions()

		// Add routes for setup tool
		makeSetupRoutes(factory, e)

		// When running the setup tool, wait a second, then open a browser window to the correct URL
		openLocalhostBrowser(factory, configOptions...)

		// Prepare HTTP (only) server using the new configuration
		go startHTTP(factory, e, configOptions...)

	} else {
		// Add routes for standard web server
		makeStandardRoutes(factory, e)

		// Prepare HTTP and HTTPS servers using the new configuration
		go startHTTP(factory, e)
		go startHTTPS(factory, e)
	}

	// Listen to the OS SIGINT channel for an interrupt signal
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	// https://golang.org/pkg/os/signal/#Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// Wait for the "quit" signal from the OS, then shut down
	<-quit
	gracefulShutdown(e)
}

/******************************************
 * Routes for Different Application Modes
 ******************************************/

// makeSetupRoutes generates a new Echo instance for the setup behavior
func makeSetupRoutes(factory *server.Factory, e *echo.Echo) {

	log.Info().Msg("Starting Emissary Setup Console")

	// Locate the setup templates
	setupFiles, err := fs.Sub(embeddedFiles, "_embed/setup")

	if err != nil {
		panic("Unable to open embedded files for setup. " + err.Error())
	}

	setupTemplates := template.Must(template.New("").
		Funcs(factory.FuncMap()).
		ParseFS(setupFiles, "*.html"))

	// Middleware for setup pages
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
	e.GET("/.themes/:themeId/:bundleId", handler.GetThemeBundle(factory))
	e.GET("/.themes/:themeId/resources/:filename", handler.GetThemeResource(factory))
}

// makeStandardRoutes generates a new Echo instance the primary server behavior
func makeStandardRoutes(factory *server.Factory, e *echo.Echo) {

	log.Info().Msg("Starting Emissary Server.")

	// WAF Middleware
	e.Pre(dome4echo.New(factory.DigitalDome()))

	e.Pre(mw.HttpsRedirect)
	e.Pre(middleware.RemoveTrailingSlash())

	// Middleware for standard pages
	// e.Use(mw.Debug()) <- this is super chatty, so only enable it on dev, or for short periods of time.
	e.Use(mw.Domain(factory))
	e.Use(steranko.Middleware(factory))
	e.Use(middleware.CORS())

	// TODO: Commonly accessed routest that we should serve
	e.GET("/robots.txt", handler.TBD)                       // https://developers.google.com/search/docs/advanced/robots/create-robots-txt
	e.GET("/sitemap.xml", handler.TBD)                      // https://developers.google.com/search/docs/advanced/sitemaps/build-sitemap
	e.GET("/humans.txt", handler.TBD)                       // http://humanstxt.org/
	e.GET("/ads.txt", handler.TBD)                          // https://iabtechlab.com/standards/ads-txt/
	e.GET("/security.txt", handler.TBD)                     // https://securitytxt.org/
	e.GET("/.well-known/security.txt", handler.TBD)         // https://securitytxt.org/
	e.GET("/.well-known/x-nodeinfo2", handler.TBD)          // Friendica polls this route
	e.GET("/poco", handler.TBD)                             // Friendica polls this route
	e.GET("/api/**", handler.TBD)                           // Mastodon API?
	e.GET("/favicon.ico", handler.TBD)                      // https://developer.mozilla.org/en-US/docs/Glossary/Favicon
	e.GET("/favicon.png", handler.TBD)                      // https://developer.mozilla.org/en-US/docs/Glossary/Favicon
	e.GET("/apple-touch-icon.png", handler.TBD)             // https://developer.apple.com/library/archive/documentation/AppleApplications/Reference/SafariWebContent/ConfiguringWebApplications/ConfiguringWebApplications.html
	e.GET("/apple-touch-icon-precomposed.png", handler.TBD) // https://developer.apple.com/library/archive/documentation/AppleApplications/Reference/SafariWebContent/ConfiguringWebApplications/ConfiguringWebApplications.html
	e.GET("/manifest.json", handler.TBD)                    // https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/manifest.json

	// TODO: MEDIUM: Add other Well-Known API calls?
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.GET("/.well-known/change-password", handler.GetChangePassword(factory))
	e.GET("/.well-known/host-meta", handler.GetHostMeta(factory))
	e.GET("/.well-known/host-meta.json", handler.GetHostMetaJSON(factory))
	e.GET("/.well-known/webfinger", handler.GetWebfinger(factory))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factory))
	e.GET("/.well-known/nodeinfo/2.0", handler.GetNodeInfo20(factory))
	e.GET("/.well-known/nodeinfo/2.1", handler.GetNodeInfo21(factory))
	e.GET("/nodeinfo/2.0", handler.GetNodeInfo20(factory))
	e.GET("/nodeinfo/2.0.json", handler.GetNodeInfo20(factory))
	e.GET("/nodeinfo/2.1", handler.GetNodeInfo21(factory))
	e.GET("/nodeinfo/2.1.json", handler.GetNodeInfo21(factory))

	// Built-In Service  Routes
	e.POST("/.follower/new", handler.PostEmailFollower(factory))
	e.GET("/.giphy", handler.GetGiphyWidget(factory))
	e.GET("/.oembed", handler.WithFactory(factory, handler.GetOEmbed))
	e.POST("/.stripe", stripe.PostWebhook(factory))
	e.GET("/.searchTag/:searchTagId/attachments/:attachmentId", handler.WithFactory(factory, handler.GetSearchTagAttachment))
	e.GET("/.themes/:themeId/:bundleId", handler.GetThemeBundle(factory))
	e.GET("/.themes/:themeId/resources/:filename", handler.GetThemeResource(factory))
	e.GET("/.templates/:templateId/:bundleId", handler.GetTemplateBundle(factory))
	e.GET("/.templates/:templateId/resources/:filename", handler.GetTemplateResource(factory))
	e.GET("/.unsplash/photos/:photo", unsplash.GetPhoto(factory))
	e.GET("/.unsplash/collections/:collection/random", unsplash.GetCollectionRandom(factory))
	e.GET("/.webmention", handler.TBD)
	e.POST("/.webmention", handler.WithFactory(factory, handler.PostWebMention))
	e.GET("/.websub/:userId/:followingId", handler.GetWebSubClient(factory))
	e.POST("/.websub/:userId/:followingId", handler.PostWebSubClient(factory))
	e.GET("/.widgets/:widgetId/:bundleId", handler.GetWidgetBundle(factory))
	e.GET("/.widgets/:widgetId/resources/:filename", handler.GetWidgetResource(factory))

	// Activity Intents
	e.GET("/.intents/discover", handler.WithFactory(factory, handler.GetIntentInfo))
	e.GET("/.intents/:intent", handler.WithFactory(factory, handler.GetOutboundIntent))
	e.POST("/.ostatus/discover", handler.PostOStatusDiscover(factory))
	e.GET("/.ostatus/tunnel", handler.GetFollowingTunnel)
	// TODO: LOW: .ostatus/tunnel is no longer necessary because we're using the right cookie settings now.
	// Migrate calls to this to a more direct route.

	// ActivityPub Routes for Search Results
	e.GET("/.search", handler.WithSearchQuery(factory, ap_search.GetJSONLD))
	e.GET("/.search/:searchId", handler.WithSearchQuery(factory, ap_search.GetJSONLD))
	e.POST("/.search/:searchId/inbox", handler.WithSearchQuery(factory, ap_search.PostInbox))
	e.GET("/.search/:searchId/outbox", handler.WithSearchQuery(factory, ap_search.GetOutboxCollection))

	// Authentication Pages
	e.GET("/signin", handler.GetSignIn(factory))
	e.POST("/signin", handler.PostSignIn(factory))
	e.POST("/signout", handler.PostSignOut(factory))
	e.GET("/register", handler.WithRegistration(factory, handler.GetRegister))
	e.GET("/register/:action", handler.WithRegistration(factory, handler.GetRegister))
	e.POST("/register", handler.WithRegistration(factory, handler.PostRegister))
	e.GET("/register/complete", handler.WithRegistration(factory, handler.GetCompleteRegistration))
	e.POST("/register/update", handler.WithRegistration(factory, handler.PostRegister))
	e.GET("/signin/reset", handler.GetResetPassword(factory))
	e.POST("/signin/reset", handler.PostResetPassword(factory))
	e.GET("/signin/reset-code", handler.GetResetCode(factory))
	e.POST("/signin/reset-code", handler.PostResetCode(factory))
	e.POST("/.masquerade", handler.PostMasquerade(factory), mw.Owner)
	e.GET("/.sso", handler.WithDomain(factory, handler.GetSingleSignOn))

	// Domain Pages
	e.GET("/.domain/attachments/:attachmentId", handler.GetDomainAttachment(factory))

	// Stream Pages
	e.GET("/", handler.WithTemplate(factory, handler.GetStream))
	e.GET("/:stream", handler.WithTemplate(factory, handler.GetStream))
	e.GET("/:stream/:action", handler.WithTemplate(factory, handler.GetStreamWithAction))
	e.POST("/:stream/:action", handler.WithTemplate(factory, handler.PostStreamWithAction))
	e.DELETE("/:stream", handler.WithTemplate(factory, handler.PostStreamWithAction))

	// Hard-coded routes for additional stream services
	e.GET("/:stream/attachments/:attachmentId", handler.GetStreamAttachment(factory)) // TODO: LOW: Can Stream Attachments be moved into a custom build step?
	e.GET("/:stream/qrcode", handler.GetQRCode(factory))                              // TODO: LOW: Can QR Codes be moved into a custom build step?
	e.GET("/:objectId/sse", handler.WithFactory(factory, handler.ServerSentEvent))
	e.GET("/@:objectId/sse", handler.WithFactory(factory, handler.ServerSentEvent))

	// Profile Pages
	// NOTE: these are rewritten from /@:userId by the rewrite middleware
	e.GET("/@service", handler.GetServiceActor(factory))
	e.POST("/@service/inbox", handler.PostServiceActor_Inbox(factory))
	e.GET("/@service/inbox", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@service/outbox", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@service/following", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@service/followers", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@service/liked", handler.WithFactory(factory, handler.GetEmptyCollection))

	// Profile Pages for "me" only routes
	e.GET("/@me", handler.WithAuthenticatedUser(factory, handler.ForwardMeURLs))
	e.POST("/@me/delete", handler.WithAuthenticatedUser(factory, handler.PostProfileDelete))

	e.GET("/@me/inbox", handler.GetInbox(factory))
	e.POST("/@me/inbox", handler.PostInbox(factory))
	e.GET("/@me/inbox/:action", handler.GetInbox(factory))
	e.POST("/@me/inbox/:action", handler.PostInbox(factory))
	e.GET("/@me/intent/create", handler.WithAuthenticatedUser(factory, handler.GetIntent_Create))
	e.POST("/@me/intent/create", handler.WithAuthenticatedUser(factory, handler.PostIntent_Create))
	e.GET("/@me/intent/dislike", handler.WithAuthenticatedUser(factory, handler.GetIntent_Dislike))
	e.POST("/@me/intent/dislike", handler.WithAuthenticatedUser(factory, handler.PostIntent_Dislike))
	e.GET("/@me/intent/follow", handler.WithAuthenticatedUser(factory, handler.GetIntent_Follow))
	e.POST("/@me/intent/follow", handler.WithAuthenticatedUser(factory, handler.PostIntent_Follow))
	e.GET("/@me/intent/like", handler.WithAuthenticatedUser(factory, handler.GetIntent_Like))
	e.POST("/@me/intent/like", handler.WithAuthenticatedUser(factory, handler.PostIntent_Like))
	e.GET("/@me/intent/continue", handler.WithAuthenticatedUser(factory, handler.GetIntent_Continue))

	// Routes for Users
	e.GET("/@:userId", handler.WithUserForwarding(factory, handler.GetOutbox))
	e.POST("/@:userId", handler.WithUser(factory, handler.PostOutbox))
	e.GET("/@:userId/:action", handler.WithUser(factory, handler.GetOutbox))
	e.POST("/@:userId/:action", handler.WithUser(factory, handler.PostOutbox))
	e.GET("/@:userId/attachments/:attachmentId", handler.GetUserAttachment(factory))

	// ActivityPub Routes for Users
	e.GET("/@:userId/pub", handler.WithUser(factory, handler.GetOutbox))
	e.POST("/@:userId/pub/inbox", ap_user.PostInbox(factory))
	e.GET("/@:userId/pub/outbox", ap_user.GetOutboxCollection(factory))
	e.GET("/@:userId/pub/followers", handler.WithFactory(factory, ap_user.GetFollowersCollection))
	e.GET("/@:userId/pub/following", handler.WithFactory(factory, ap_user.GetFollowingCollection))
	e.GET("/@:userId/pub/following/:followingId", handler.WithFactory(factory, ap_user.GetFollowingRecord))
	e.GET("/@:userId/pub/shared", ap_user.GetResponseCollection(factory, vocab.ActivityTypeAnnounce))
	e.GET("/@:userId/pub/shared/:response", ap_user.GetResponse(factory, vocab.ActivityTypeAnnounce))
	e.GET("/@:userId/pub/liked", ap_user.GetResponseCollection(factory, vocab.ActivityTypeLike))
	e.GET("/@:userId/pub/liked/:response", ap_user.GetResponse(factory, vocab.ActivityTypeLike))
	e.GET("/@:userId/pub/disliked", ap_user.GetResponseCollection(factory, vocab.ActivityTypeDislike))
	e.GET("/@:userId/pub/disliked/:response", ap_user.GetResponse(factory, vocab.ActivityTypeDislike))
	e.GET("/@:userId/pub/blocked", ap_user.GetBlockedCollection(factory))
	e.GET("/@:userId/pub/blocked/:ruleId", ap_user.GetBlock(factory))

	// ActivityPub Routes for Streams
	e.GET("/:stream/pub", handler.WithTemplate(factory, ap_stream.GetJSONLD))
	e.POST("/:stream/pub/inbox", ap_stream.PostInbox(factory))
	e.GET("/:stream/pub/outbox", ap_stream.GetOutboxCollection(factory))
	e.GET("/:stream/pub/followers", ap_stream.GetFollowersCollection(factory))
	e.GET("/:stream/pub/children", handler.WithFactory(factory, ap_stream.GetChildrenCollection))

	// Domain Admin Pages
	e.GET("/admin", handler.GetAdmin(factory), mw.Owner)
	e.GET("/admin/:param1", handler.GetAdmin(factory), mw.Owner)
	e.POST("/admin/:param1", handler.PostAdmin(factory), mw.Owner)
	e.GET("/admin/:param1/:param2", handler.GetAdmin(factory), mw.Owner)
	e.POST("/admin/:param1/:param2", handler.PostAdmin(factory), mw.Owner)
	e.GET("/admin/:param1/:param2/:param3", handler.GetAdmin(factory), mw.Owner)
	e.POST("/admin/:param1/:param2/:param3", handler.PostAdmin(factory), mw.Owner)
	e.POST("/admin/index-all-streams", handler.WithFactory(factory, handler.IndexAllStreams), mw.Owner)
	e.POST("/admin/index-all-users", handler.WithFactory(factory, handler.IndexAllUsers), mw.Owner)

	// OAuth Client Connections
	e.GET("/oauth/clients/:provider", handler.GetOAuth(factory), mw.Owner)
	e.GET("/oauth/clients/:provider/callback", handler.GetOAuthCallback(factory), mw.AllowCSR, mw.Owner)
	e.GET("/oauth/clients/redirect", handler.OAuthRedirect(factory), mw.Owner)

	// Startup Wizard
	e.GET("/startup", handler.GetStartup(factory), mw.Owner)
	e.GET("/startup/:action", handler.GetStartup(factory), mw.Owner)
	e.POST("/startup", handler.PostStartup(factory), mw.Owner)

	// OAuth Server
	e.GET("/oauth/authorize", handler.GetOAuthAuthorization(factory), mw.Authenticated)
	e.POST("/oauth/authorize", handler.PostOAuthAuthorization(factory), mw.Authenticated)
	e.POST("/oauth/token", handler.PostOAuthToken(factory))
	e.POST("/oauth/revoke", handler.PostOAuthRevoke(factory))

	// Mastodon API
	// toot.Register(e, handler.Mastodon(factory))
}

/******************************************
 * Additional Helper Functions
 ******************************************/

// openLocalhostBrowser opens a browser window to the localhost URL
// IF the server is configured to run on HTTP or HTTPS
func openLocalhostBrowser(factory *server.Factory, options ...config.Option) {

	// Get and modify the configuration
	config := factory.Config()
	config.With(options...)

	if portString, ok := config.HTTPPortString(); ok {
		time.Sleep(500 * time.Millisecond)

		if err := browser.OpenURL("http://localhost" + portString + "/"); err != nil {
			log.Debug().Err(err).Msg("Unable to open setup tool browser window. Visit http://localhost" + portString + "/ in your web browser to edit Emissary settings")
		}

	} else {
		log.Error().Msg("Unable to open setup tool because no HTTP port is configured.")
		os.Exit(0)
	}
}

// startHTTP starts the HTTPS server using Let's Encrypt SSL certificates.
// If the configured port is not available, it will wait one second and retry until it is
func startHTTPS(factory *server.Factory, e *echo.Echo, options ...config.Option) {

	// Get and modify the configuration
	config := factory.Config()
	config.With(options...)

	// If HTTPS is configured, then try to start an HTTPS server
	if portString, ok := config.HTTPSPortString(); ok {

		// Find all NON-LOCAL domain names.  We need AT LEAST ONE to get an SSL Certificate
		domains := slice.Filter(config.DomainNames(), domain.NotLocalhost)

		if len(domains) == 0 {
			log.Info().Msg("Skipping HTTPS server because there are no non-local domains.")
			return
		}

		// Initialize Let's Encrypt autocert for TLS certificates
		e.AutoTLSManager = autocert.Manager{
			HostPolicy: autocert.HostWhitelist(domains...),
			Cache:      autocert.DirCache(config.Certificates["location"]),
			Prompt:     autocert.AcceptTOS,
			Email:      config.AdminEmail,
		}

		log.Info().Msg("Starting HTTPS server on port " + portString + ".")

		for {
			if err := e.StartAutoTLS(portString); err != nil {
				log.Error().Err(err).Send()
				time.Sleep(1 * time.Second)
			}
		}
	}

	log.Info().Msg("NO HTTPS PORT CONFIGURED. Skipping HTTPS server.")
}

// startHTTP starts the HTTP server.
// If the configured port is not available, it will wait one second and retry until it is
func startHTTP(factory *server.Factory, e *echo.Echo, options ...config.Option) {

	// Get and modify the configuration
	config := factory.Config()
	config.With(options...)

	if portString, ok := config.HTTPPortString(); ok {

		log.Info().Msg("Starting HTTP server on port " + portString + ".")

		for {
			if err := e.Start(portString); err != nil {
				log.Error().Err(err).Send()
				time.Sleep(1 * time.Second)
			}
		}
	}

	log.Info().Msg("NO HTTP PORT CONFIGURED. Skipping HTTP server")
}

func waitForSigInt() {
	// Listen to the OS SIGINT channel for an interrupt signal
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	// https://golang.org/pkg/os/signal/#Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// Wait for an interrupt signal from the OS
	<-quit

	// Shut down the echo server gracefully
	// gracefulShutdown(e)

	// Exit the program (forcefully)
	os.Exit(0)
}

// gracefulShutdown listens for a SIGINT signal, then shuts down the server gracefully
func gracefulShutdown(e *echo.Echo) {

	// Get a cancellation context with a 5 second timeout
	// https://echo.labstack.com/docs/cookbook/graceful-shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to shut down the server
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

// errorHandler is a custom error handler that returns a JSON error message to the client
func errorHandler(err error, ctx echo.Context) {

	// Special handling of permisssion errors
	request := ctx.Request()

	errorCode := derp.ErrorCode(err)

	switch errorCode {

	case http.StatusNotFound:
		_ = ctx.String(derp.ErrorCode(err), derp.Message(err))
		return

	case http.StatusUnauthorized:

		uri := request.URL

		if currentPath := uri.Path; currentPath != "/signin" {
			nextPage := uri.String()
			_ = ctx.Redirect(http.StatusSeeOther, "/signin?next="+url.QueryEscape(nextPage))
			return
		}

		_ = ctx.String(errorCode, derp.Message(err))
		return
	}

	// Write the error to the console (on production and local domains)
	derp.Report(err)

	// On localhost, allow developers to see full error dump.
	if domain.IsLocalhost(ctx.Request().Host) {
		_ = ctx.JSONPretty(errorCode, err, "  ")
		return
	}

	// Fall through to general error handler
	_ = ctx.String(derp.ErrorCode(err), derp.Message(err))
}
