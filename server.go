package main

import (
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4/middleware"
	"github.com/whisperverse/whisperverse/config"
	"github.com/whisperverse/whisperverse/route"
	"github.com/whisperverse/whisperverse/server"
)

func main() {

	spew.Config.DisableMethods = true

	fmt.Println("Starting GHOST")
	fmt.Println("Loading configuration file...")

	c, err := config.Load("./config.json")

	if err != nil {
		derp.Report(err)
		return
	}

	fmt.Println("Initializing hosts...")

	factory := server.NewFactory(c)

	fmt.Println("Initializing web server...")
	e := route.New(factory)

	e.Use(middleware.Recover())
	// TODO: implement echo.Security middleware
	e.Use(steranko.Middleware(factory))

	/*
		e.AutoTLSManager = autocert.Manager{
			HostPolicy: autocert.HostWhitelist(c.DomainNames()...),
			Cache:      autocert.DirCache(".cache"),
			Prompt:     autocert.AcceptTOS,
		}

		fmt.Println("Starting web server..")
		e.Logger.Fatal(e.StartAutoTLS(":443"))
	*/

	fmt.Println("Starting web server..")
	e.Logger.Fatal(e.Start(":80"))
}
