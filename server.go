package main

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	mw "github.com/EmissarySocial/emissary/middleware"
	"github.com/EmissarySocial/emissary/route"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/setup"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

//go:embed all:_static/*
var staticFiles embed.FS

func main() {

	fmt.Println("Starting Emissary.")

	// Set global configuration
	spew.Config.DisableMethods = true

	// See if the setup command is being run
	commandLineArgs := config.GetCommandLineArgs()

	configStorage := config.Load(commandLineArgs)

	// Special case to execute the setup server
	if commandLineArgs.Setup {
		fmt.Println("Starting Setup Mode...")
		setup.Setup(configStorage, staticFiles)
		return
	}

	// FALL THROUGH means we're running the standard server
	var e *echo.Echo

	// Every time the configuration is updated, create a new server (and swap the old one, if necessary)
	for c := range configStorage.Subscribe() {

		fmt.Println("Reading configuration file...")

		factory := server.NewFactory(configStorage)
		domains := c.DomainNames()

		fmt.Println("Setting up new server on " + convert.String(len(domains)) + " domains: " + strings.Join(domains, ", "))

		newServer := route.New(factory)

		// Global middleware
		// TODO: implement echo.Security middleware
		newServer.Use(middleware.Recover())
		newServer.Use(mw.HttpsRedirect)
		newServer.Use(steranko.Middleware(factory))

		// Prepare HTTP and HTTPS servers using the new configuration
		go startHttps(newServer, c)
		go startHttp(newServer)

		// If there is already a server running, then do a graceful shutdown
		if e != nil {

			// Context for graceful shutdown
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			// Try to shut down the server
			if err := e.Shutdown(ctx); err != nil {
				derp.Report(err)
			}
			cancel()
		}

		// Save 'newServer' as the currently running server
		e = newServer
	}
}

func startHttps(e *echo.Echo, c config.Config) {

	// Find all NON-LOCAL domain names
	domains := slice.Filter(c.DomainNames(), func(v string) bool {
		if v == "localhost" {
			return false
		}

		if strings.HasSuffix(v, ".local") {
			return false
		}

		if strings.HasPrefix(v, "10.") {
			return false
		}

		if strings.HasPrefix(v, "192.168") {
			return false
		}

		return true
	})

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

	for {
		if err := e.StartAutoTLS(":443"); err != nil {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func startHttp(e *echo.Echo) {
	fmt.Println("Starting HTTP server...")
	for {
		if err := e.Start(":80"); err != nil {
			time.Sleep(10 * time.Millisecond)
		}
	}
}
