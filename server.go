package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/config"
	mw "github.com/EmissarySocial/emissary/middleware"
	"github.com/EmissarySocial/emissary/route"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

func main() {

	spew.Config.DisableMethods = true

	fmt.Println("Starting Whisperverse.")
	fmt.Println("Loading configuration file...")

	c := config.Load()

	fmt.Println("Initializing hosts...")

	factory := server.NewFactory(c)

	fmt.Println("Initializing web server...")
	e := route.New(factory)

	// Global middleware
	// TODO: implement echo.Security middleware
	e.Use(middleware.Recover())
	e.Use(mw.HttpsRedirect)
	e.Use(steranko.Middleware(factory))

	go startHttps(e, c) // Start HTTPS server in background, so that...
	startHttp(e)        // ...we can also start the HTTP server
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

	fmt.Print("Starting HTTPS web server for:")
	for _, domain := range domains {
		fmt.Print(" " + domain)
	}
	fmt.Println("")

	// Initialize Let's Encrypt autocert for TLS certificates
	e.AutoTLSManager = autocert.Manager{
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache(c.Certificates.Location),
		Prompt:     autocert.AcceptTOS,
		Email:      c.AdminEmail,
	}

	for {
		if err := e.StartAutoTLS(":443"); err != nil {
			fmt.Print(".")
			time.Sleep(1 * time.Second)
		}
	}
}

func startHttp(e *echo.Echo) {
	fmt.Println("Starting HTTP web server: ")
	for {
		if err := e.Start(":80"); err != nil {
			fmt.Print(".")
			time.Sleep(1 * time.Second)
		}
	}
}
