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
	ap_domain "github.com/EmissarySocial/emissary/handler/activitypub_domain"
	ap_search "github.com/EmissarySocial/emissary/handler/activitypub_search"
	ap_stream "github.com/EmissarySocial/emissary/handler/activitypub_stream"
	ap_user "github.com/EmissarySocial/emissary/handler/activitypub_user"
	"github.com/EmissarySocial/emissary/handler/stripe"
	"github.com/EmissarySocial/emissary/handler/unsplash"
	mw "github.com/EmissarySocial/emissary/middleware"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	derpconsole "github.com/EmissarySocial/emissary/tools/derp-console"
	"github.com/benpate/derp"
	"github.com/benpate/digital-dome/dome4echo"
	dt "github.com/benpate/domain"
	"github.com/benpate/form/widget"
	"github.com/benpate/rosetta/slice"
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

	// Derp configuration (rewritten once we have a shared database)
	derp.Plugins.Clear()
	derp.Plugins.Add(derpconsole.New())

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

	if serverFactory := server.NewFactory(&commandLineArgs, embeddedFiles); serverFactory.IsSetupMode() {

		// Get config modifiers from the command line (like HTTP PORT)
		configOptions := commandLineArgs.ConfigOptions()

		// Add routes for setup tool
		makeSetupRoutes(serverFactory, e)

		// When running the setup tool, wait a second, then open a browser window to the correct URL
		openLocalhostBrowser(serverFactory, configOptions...)

		// Prepare HTTP (only) server using the new configuration
		go startHTTP(serverFactory, e, configOptions...)

	} else {

		// Add routes for standard web server
		makeStandardRoutes(serverFactory, e)

		// Prepare HTTP and HTTPS servers using the new configuration
		go startHTTP(serverFactory, e)
		go startHTTPS(serverFactory, e)
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
	e.GET("/config", handler.SetupGetConfig(factory))
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
	// e.Use(steranko.Middleware(factory))
	e.Use(middleware.CORS())

	// TODO: Commonly accessed routest that we should serve
	e.GET("/robots.txt", handler.RobotsTxt)                 // https://developers.google.com/search/docs/advanced/robots/create-robots-txt
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

	// NodeInfo Routes
	e.GET("/nodeinfo/2.0", handler.WithFactory(factory, handler.GetNodeInfo20))
	e.GET("/nodeinfo/2.0.json", handler.WithFactory(factory, handler.GetNodeInfo20))
	e.GET("/nodeinfo/2.1", handler.WithFactory(factory, handler.GetNodeInfo21))
	e.GET("/nodeinfo/2.1.json", handler.WithFactory(factory, handler.GetNodeInfo21))

	// Built-In Service Routes
	e.GET("/.api/actors", handler.WithAuthenticatedAPI(factory, handler.GetAPIActors))
	e.GET("/.checkout", handler.WithProduct(factory, handler.GetCheckout))
	e.GET("/.checkout/response", handler.WithMerchantAccountJWT(factory, handler.GetCheckoutResponse))
	e.POST("/.follower/new", handler.WithFactory(factory, handler.PostEmailFollower))
	e.GET("/.geocode/network", handler.WithFactory(factory, handler.GetGeocodeNetwork))
	e.GET("/.geocode/autocomplete", handler.WithFactory(factory, handler.GetGeocodeAutocomplete))
	e.GET("/.giphy", handler.WithFactory(factory, handler.GetGiphyWidget))
	e.GET("/.imported", handler.WithFactory(factory, handler.GetImportedURL))
	e.GET("/.intents/discover", handler.WithFactory(factory, handler.GetIntentInfo))
	e.GET("/.intents/:intent", handler.WithFactory(factory, handler.GetOutboundIntent))
	e.POST("/.masquerade", handler.WithOwner(factory, handler.PostMasquerade))
	e.GET("/.oembed", handler.WithFactory(factory, handler.GetOEmbed))
	e.POST("/.ostatus/discover", handler.WithFactory(factory, handler.PostOStatusDiscover))
	e.GET("/.ostatus/tunnel", handler.GetFollowingTunnel)
	e.GET("/.searchTag/:searchTagId/attachments/:attachmentId", handler.WithFactory(factory, handler.GetSearchTagAttachment))
	e.GET("/.sso", handler.WithDomain(factory, handler.GetSingleSignOn))
	e.POST("/.stripe/webhook/signup", handler.WithDomain(factory, stripe.PostSignupWebhook))
	e.POST("/.stripe/webhook/checkout", handler.WithMerchantAccount(factory, handler.PostStripeWebhook_Checkout))
	e.GET("/.stripe-connect/connect", handler.WithAuthenticatedUser(factory, handler.GetStripeConnect))
	e.POST("/.stripe-connect/webhook/signup", handler.WithDomain(factory, stripe.PostSignupWebhook))
	e.POST("/.stripe-connect/webhook/checkout", handler.WithConnection(model.ConnectionProviderStripeConnect, factory, handler.PostStripeConnectWebhook_Checkout))
	e.GET("/.themes/:themeId/:bundleId", handler.GetThemeBundle(factory))
	e.GET("/.themes/:themeId/resources/:filename", handler.GetThemeResource(factory))
	e.GET("/.templates/:templateId/:bundleId", handler.GetTemplateBundle(factory))
	e.GET("/.templates/:templateId/resources/:filename", handler.GetTemplateResource(factory))
	e.GET("/.unsplash/photos/:photo", handler.WithFactory(factory, unsplash.GetPhoto))
	e.GET("/.unsplash/collections/:collection/random", handler.WithFactory(factory, unsplash.GetCollectionRandom))
	e.GET("/.validate/username", handler.WithFactory(factory, handler.GetValidateUsername))
	e.GET("/.webmention", handler.TBD)
	e.POST("/.webmention", handler.WithFactory(factory, handler.PostWebMention))
	e.GET("/.websub/:userId/:followingId", handler.WithFactory(factory, handler.GetWebSubClient))
	e.POST("/.websub/:userId/:followingId", handler.WithFactory(factory, handler.PostWebSubClient))
	e.GET("/.widgets/:widgetId/:bundleId", handler.GetWidgetBundle(factory))
	e.GET("/.widgets/:widgetId/resources/:filename", handler.GetWidgetResource(factory))

	// Well-Known Routes https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers
	e.GET("/.well-known/change-password", handler.GetChangePassword(factory))
	e.GET("/.well-known/host-meta", handler.GetHostMeta(factory))
	e.GET("/.well-known/host-meta.json", handler.GetHostMetaJSON(factory))
	e.GET("/.well-known/webfinger", handler.WithFactory(factory, handler.GetWebfinger))
	e.GET("/.well-known/nodeinfo", handler.WithFactory(factory, handler.GetNodeInfo))
	e.GET("/.well-known/nodeinfo/2.0", handler.WithFactory(factory, handler.GetNodeInfo20))
	e.GET("/.well-known/nodeinfo/2.1", handler.WithFactory(factory, handler.GetNodeInfo21))
	e.GET("/.well-known/oauth-authorization-server", handler.TBD)

	// Authentication Pages
	e.GET("/signin", handler.WithFactory(factory, handler.GetSignIn))
	e.POST("/signin", handler.WithFactory(factory, handler.PostSignIn))
	e.GET("/signout", handler.WithFactory(factory, handler.GetSignOut))
	e.POST("/signout", handler.WithFactory(factory, handler.PostSignOut))
	e.GET("/register", handler.WithRegistration(factory, handler.GetRegister))
	e.GET("/register/:action", handler.WithRegistration(factory, handler.GetRegister))
	e.POST("/register", handler.WithRegistration(factory, handler.PostRegister))
	e.GET("/register/complete", handler.WithRegistration(factory, handler.GetCompleteRegistration))
	e.POST("/register/update", handler.WithRegistration(factory, handler.PostRegister))
	e.GET("/signin/reset", handler.WithFactory(factory, handler.GetResetPassword))
	e.POST("/signin/reset", handler.WithFactory(factory, handler.PostResetPassword))
	e.GET("/signin/reset-code", handler.WithFactory(factory, handler.GetResetCode))
	e.POST("/signin/reset-code", handler.WithFactory(factory, handler.PostResetCode))

	// Domain Pages
	e.GET("/.domain/attachments/:attachmentId", handler.WithFactory(factory, handler.GetDomainAttachment))

	// Stream Pages
	e.GET("/", handler.WithTemplate(factory, handler.GetStream))
	e.GET("/:stream", handler.WithTemplate(factory, handler.GetStream))
	e.GET("/:stream/:action", handler.WithTemplate(factory, handler.GetStreamWithAction))
	e.POST("/:stream/:action", handler.WithTemplate(factory, handler.PostStreamWithAction))
	e.DELETE("/:stream", handler.WithTemplate(factory, handler.PostStreamWithAction))

	// Hard-coded routes for additional stream services
	e.GET("/:stream/attachments/:attachmentId", handler.WithStream(factory, handler.GetStreamAttachment))
	e.GET("/:stream/qrcode", handler.WithStream(factory, handler.GetQRCode_Stream))
	e.GET("/:objectId/sse", handler.WithFactory(factory, handler.ServerSentEvent))
	e.GET("/:objectId/sse/updated", handler.WithFactory(factory, handler.ServerSentEvent_Updated))
	e.GET("/:objectId/sse/child-updated", handler.WithFactory(factory, handler.ServerSentEvent_ChildUpdated))
	e.GET("/:objectId/sse/new-replies", handler.WithFactory(factory, handler.ServerSentEvent_NewReplies))
	e.GET("/:objectId/sse/import-progress", handler.WithFactory(factory, handler.ServerSentEvent_ImportProgress))
	e.GET("/@:objectId/sse", handler.WithFactory(factory, handler.ServerSentEvent))
	e.GET("/@:objectId/sse/updated", handler.WithFactory(factory, handler.ServerSentEvent_Updated))

	// ActivityPub pages for the application actor
	e.GET("/@application", handler.WithFactory(factory, handler.GetApplicationActor))
	e.POST("/@application/inbox", handler.PostApplicationActor_Inbox(factory))
	e.GET("/@application/inbox", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@application/outbox", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@application/following", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@application/followers", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@application/liked", handler.WithFactory(factory, handler.GetEmptyCollection))

	// Profile Pages for "me" only routes
	e.GET("/@me", handler.WithAuthenticatedUser(factory, handler.ForwardMeURLs))
	e.POST("/@me/delete", handler.WithAuthenticatedUser(factory, handler.PostProfileDelete))

	e.GET("/@me/inbox", handler.WithAuthenticatedUser(factory, handler.GetInbox))
	e.POST("/@me/inbox", handler.WithAuthenticatedUser(factory, handler.PostInbox))
	e.GET("/@me/inbox/:action", handler.WithAuthenticatedUser(factory, handler.GetInbox))
	e.POST("/@me/inbox/:action", handler.WithAuthenticatedUser(factory, handler.PostInbox))
	e.GET("/@me/settings", handler.WithAuthenticatedUser(factory, handler.GetSettings))
	e.POST("/@me/settings", handler.WithAuthenticatedUser(factory, handler.PostSettings))
	e.GET("/@me/settings/:action", handler.WithAuthenticatedUser(factory, handler.GetSettings))
	e.POST("/@me/settings/:action", handler.WithAuthenticatedUser(factory, handler.PostSettings))
	e.GET("/@me/outbox", handler.WithAuthenticatedUser(factory, ap_user.GetOutboxCollection))
	e.POST("/@me/outbox", handler.WithAuthenticatedUser(factory, ap_user.PostOutbox))

	e.GET("/@me/intent/create", handler.WithAuthenticatedUser(factory, handler.GetIntent_Create))
	e.POST("/@me/intent/create", handler.WithAuthenticatedUser(factory, handler.PostIntent_Create))
	e.GET("/@me/intent/dislike", handler.WithAuthenticatedUser(factory, handler.GetIntent_Dislike))
	e.POST("/@me/intent/dislike", handler.WithAuthenticatedUser(factory, handler.PostIntent_Dislike))
	e.GET("/@me/intent/follow", handler.WithAuthenticatedUser(factory, handler.GetIntent_Follow))
	e.POST("/@me/intent/follow", handler.WithAuthenticatedUser(factory, handler.PostIntent_Follow))
	e.GET("/@me/intent/like", handler.WithAuthenticatedUser(factory, handler.GetIntent_Like))
	e.POST("/@me/intent/like", handler.WithAuthenticatedUser(factory, handler.PostIntent_Like))
	e.GET("/@me/intent/continue", handler.WithAuthenticatedUser(factory, handler.GetIntent_Continue))

	e.GET("/@guest", handler.WithIdentity(factory, handler.GetIdentity))
	e.POST("/@guest", handler.WithIdentity(factory, handler.PostIdentity))
	e.GET("/@guest/delete/:privilegeId", handler.WithPrivilege(factory, handler.GetPrivilegeDelete))
	e.POST("/@guest/delete/:privilegeId", handler.WithPrivilege(factory, handler.PostPrivilegeDelete))
	e.GET("/@guest/:action", handler.WithIdentity(factory, handler.GetIdentity))
	e.POST("/@guest/:action", handler.WithIdentity(factory, handler.PostIdentity))
	e.GET("/@guest/signin", handler.WithFactory(factory, handler.GetIdentitySignin))
	e.POST("/@guest/signin", handler.WithFactory(factory, handler.PostIdentitySignin))
	e.GET("/@guest/signin/:jwt", handler.WithFactory(factory, handler.GetIdentitySigninWithJWT))
	e.POST("/@guest/identifier", handler.WithIdentity(factory, handler.PostIdentityIdentifier))

	// Global Search Actor (ActivityPub)
	e.GET("/@search", handler.WithFactory(factory, ap_domain.GetJSONLD))
	e.POST("/@search/pub/followers", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.POST("/@search/pub/following", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.POST("/@search/pub/inbox", handler.WithFactory(factory, ap_domain.PostInbox))
	e.GET("/@search/pub/outbox", handler.WithFactory(factory, ap_domain.GetOutboxCollection))
	e.GET("/@search/pub/outbox/:searchResultId", handler.WithFactory(factory, ap_domain.GetOutboxMessage))

	// Search Query Routes (ActivityPub)
	e.POST("/.searchQuery", handler.WithFactory(factory, handler.PostSearchLookup))
	e.GET("/@search_:searchId", handler.WithSearchQuery(factory, ap_search.GetJSONLD))
	e.POST("/@search_:searchId/pub/followers", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.POST("/@search_:searchId/pub/following", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.GET("/@search_:searchId/pub/inbox", handler.WithFactory(factory, handler.GetEmptyCollection))
	e.POST("/@search_:searchId/pub/inbox", handler.WithSearchQuery(factory, ap_search.PostInbox))
	e.GET("/@search_:searchId/pub/outbox", handler.WithSearchQuery(factory, ap_search.GetOutboxCollection))
	e.GET("/@search_:searchId/pub/outbox/:searchResultId", handler.WithSearchQuery(factory, ap_search.GetOutboxMessage))

	// Routes for Users
	e.GET("/@:userId", handler.WithUserForwarding(factory, handler.GetOutbox))
	e.POST("/@:userId", handler.WithUser(factory, handler.PostOutbox))
	e.GET("/@:userId/:action", handler.WithUser(factory, handler.GetOutbox))
	e.POST("/@:userId/:action", handler.WithUser(factory, handler.PostOutbox))
	e.GET("/@:userId/attachments/:attachmentId", handler.WithUser(factory, handler.GetUserAttachment))
	e.GET("/@:userId/qrcode", handler.WithUser(factory, handler.GetQRCode_User))

	// Export Routes for Users
	e.GET("/@:userId/export", handler.NotFound)
	e.POST("/@:userId/export/start", handler.WithOAuthUser(factory, handler.PostUserExportStart))
	e.POST("/@:userId/export/finish", handler.WithAuthenticatedUser(factory, handler.PostUserExportFinish))
	e.GET("/@:userId/export/complete", handler.WithFactory(factory, handler.GetUserExportComplete))
	e.GET("/@:userId/export/:collection", handler.WithOAuthUser(factory, handler.GetUserExportCollection))
	e.GET("/@:userId/export/:collection/:recordId", handler.WithOAuthUser(factory, handler.GetUserExportDocument))
	e.GET("/@:userId/export/emissary-stream/:streamId/attachments", handler.WithOAuthUserStream(factory, handler.GetAttachmentsExportCollection))
	e.GET("/@:userId/export/emissary-stream/:streamId/attachments/:recordId", handler.WithOAuthUserStream(factory, handler.GetAttachmentsExportDocument))

	// ActivityPub Routes for Users
	e.GET("/@:userId/pub", handler.WithUser(factory, handler.GetOutbox))
	e.POST("/@:userId/pub/inbox", handler.WithUser(factory, ap_user.PostInbox))
	e.GET("/@:userId/pub/outbox", handler.WithUser(factory, ap_user.GetOutboxCollection))
	e.GET("/@:userId/pub/outbox/:messageId", handler.WithUser(factory, ap_user.GetOutboxActivity))
	e.GET("/@:userId/pub/featured", handler.WithUser(factory, ap_user.GetFeaturedCollection))
	e.GET("/@:userId/pub/followers", handler.WithUser(factory, ap_user.GetFollowersCollection))
	e.GET("/@:userId/pub/following", handler.WithUser(factory, ap_user.GetFollowingCollection))
	e.GET("/@:userId/pub/following/:followingId", handler.WithUser(factory, ap_user.GetFollowingRecord))
	e.GET("/@:userId/pub/keyPackages", handler.WithUser(factory, ap_user.GetKeyPackageCollection))
	e.GET("/@:userId/pub/keyPackages/:keyPackageId", handler.WithUser(factory, ap_user.GetKeyPackageRecord))
	e.GET("/@:userId/pub/shared", handler.WithUser(factory, ap_user.GetResponseCollection))
	e.GET("/@:userId/pub/shared/:response", handler.WithUser(factory, ap_user.GetResponse))
	e.GET("/@:userId/pub/liked", handler.WithUser(factory, ap_user.GetResponseCollection))
	e.GET("/@:userId/pub/liked/:response", handler.WithUser(factory, ap_user.GetResponse))
	e.GET("/@:userId/pub/disliked", handler.WithUser(factory, ap_user.GetResponseCollection))
	e.GET("/@:userId/pub/disliked/:response", handler.WithUser(factory, ap_user.GetResponse))
	e.GET("/@:userId/pub/blocked", handler.WithUser(factory, ap_user.GetBlockedCollection))
	e.GET("/@:userId/pub/blocked/:ruleId", handler.WithUser(factory, ap_user.GetBlock))

	// ActivityPub Routes for Streams
	e.GET("/:stream/pub", handler.WithTemplate(factory, ap_stream.GetJSONLD))
	e.POST("/:stream/pub/inbox", handler.WithTemplate(factory, ap_stream.PostInbox))
	e.GET("/:stream/pub/outbox", handler.WithTemplate(factory, ap_stream.GetOutboxCollection))
	e.GET("/:stream/pub/followers", handler.WithTemplate(factory, ap_stream.GetFollowersCollection))
	e.GET("/:stream/pub/children", handler.WithStream(factory, ap_stream.GetChildrenCollection))

	// Domain Admin Pages
	e.GET("/admin", handler.WithOwner(factory, handler.GetAdmin))
	e.GET("/admin/:param1", handler.WithOwner(factory, handler.GetAdmin))
	e.POST("/admin/:param1", handler.WithOwner(factory, handler.PostAdmin))
	e.GET("/admin/:param1/:param2", handler.WithOwner(factory, handler.GetAdmin))
	e.POST("/admin/:param1/:param2", handler.WithOwner(factory, handler.PostAdmin))
	e.GET("/admin/:param1/:param2/:param3", handler.WithOwner(factory, handler.GetAdmin))
	e.POST("/admin/:param1/:param2/:param3", handler.WithOwner(factory, handler.PostAdmin))
	e.POST("/admin/reindex-activitystream-cache", handler.WithOwner(factory, handler.ReIndexActivityStreamCache))
	e.POST("/admin/index-all-streams", handler.WithOwner(factory, handler.IndexAllStreams))
	e.POST("/admin/index-all-users", handler.WithOwner(factory, handler.IndexAllUsers))

	// Startup Wizard
	e.GET("/startup", handler.WithOwner(factory, handler.GetStartup))
	e.GET("/startup/:action", handler.WithOwner(factory, handler.GetStartup))
	e.POST("/startup", handler.WithOwner(factory, handler.PostStartup))

	// OAuth Client Connections
	e.GET("/oauth/metadata", handler.WithFactory(factory, handler.GetOAuthMetadata))
	e.GET("/oauth/clients/:provider", handler.WithOwner(factory, handler.GetOAuth))
	e.GET("/oauth/clients/:provider/callback", handler.WithOwner(factory, handler.GetOAuthCallback), mw.AllowCSR)
	e.GET("/oauth/clients/import/callback", handler.WithAuthenticatedUser(factory, handler.GetOAuthImportCallback))
	e.GET("/oauth/clients/redirect", handler.WithOwner(factory, handler.OAuthRedirect))

	// OAuth Server
	e.GET("/oauth/authorize", handler.WithAuthenticatedUser(factory, handler.GetOAuthAuthorization))
	e.POST("/oauth/authorize", handler.WithAuthenticatedUser(factory, handler.PostOAuthAuthorization))
	e.POST("/oauth/token", handler.WithFactory(factory, handler.PostOAuthToken))
	e.POST("/oauth/revoke", handler.WithFactory(factory, handler.PostOAuthRevoke))

	// Mastodon API
	// toot.Register(e, handler.Mastodon(factory))
}

/******************************************
 * Start HTTP/HTTPS Servers
 ******************************************/

// startHTTP starts the HTTPS server using Let's Encrypt SSL certificates.
// If the configured port is not available, it will wait one second and retry until it is
func startHTTPS(factory *server.Factory, e *echo.Echo, options ...config.Option) {

	// Get and modify the configuration
	config := factory.Config()
	config.With(options...)

	// If HTTPS is configured, then try to start an HTTPS server
	if portString, ok := config.HTTPSPortString(); ok {

		// Find all NON-LOCAL domain names.  We need AT LEAST ONE to get an SSL Certificate
		domains := slice.Filter(config.DomainNames(), dt.NotLocalhost)

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

	const location = "server.errorHandler"

	// Special handling of permisssion errors
	request := ctx.Request()

	// Forward "Unauthorized" requests to the signin page
	if derp.IsUnauthorized((err)) {

		uri := request.URL

		if currentPath := uri.Path; currentPath != "/signin" {
			nextPage := uri.String()
			_ = ctx.Redirect(http.StatusSeeOther, "/signin?next="+url.QueryEscape(nextPage))
			return
		}

		_ = ctx.String(derp.ErrorCode(err), derp.Message(err))
		return
	}

	// Write the error to the console (on production and local domains)
	derp.Report(
		derp.Wrap(
			err,
			location,
			"Unable to generate web page",
			"url: "+dt.AddProtocol(request.Host)+request.URL.String(),
			"method: "+request.Method,
			ctx.Request().Header,
		),
	)

	// If this is a local request, then show developers a full error dump
	if hostname := dt.TrueHostname(request); dt.IsLocalhost(hostname) {
		_ = ctx.JSONPretty(derp.ErrorCode(err), err, "  ")
		return
	}

	// Fall through to general error handler that only returns the error code and message
	_ = ctx.String(derp.ErrorCode(err), derp.Message(err))
}
