package main

import (
	"fmt"

	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4/middleware"
	"github.com/whisperverse/whisperverse/config"
	"github.com/whisperverse/whisperverse/route"
	"github.com/whisperverse/whisperverse/server"
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
	e.Use(steranko.Middleware(factory))

	// Initialize Let's Encrypt autocert for TLS certificates
	e.AutoTLSManager = autocert.Manager{
		HostPolicy: autocert.HostWhitelist(c.DomainNames()...),
		Cache:      autocert.DirCache(c.Certificates.Location),
		Prompt:     autocert.AcceptTOS,
		Email:      c.AdminEmail,
	}

	spew.Dump(c.DomainNames())

	fmt.Println("Starting HTTPS web server..")
	go e.StartAutoTLS(":443")

	// Start HTTP web server
	fmt.Println("Starting HTTP web server..")
	e.Logger.Fatal(e.Start(":80"))
}
