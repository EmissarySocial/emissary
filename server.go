package main

import (
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/routes"
	"github.com/benpate/ghost/service"
	"github.com/davecgh/go-spew/spew"
)

func main() {

	fmt.Println("Starting GHOST")
	fmt.Println("Loading configuration file...")

	c, err := config.Load("./config.json")

	// Debugging for spew
	spew.Config.DisableMethods = true

	if err != nil {
		derp.Report(err)
		return
	}

	fmt.Println("Initializing hosts...")

	factoryManager := service.NewFactoryManager(c)

	if factoryManager.DomainCount() == 0 {
		fmt.Println("No Domains Configured!!")
		return
	}

	fmt.Println("Initializing web server...")
	e := routes.New(factoryManager)

	fmt.Println("Starting web server..")
	e.Logger.Fatal(e.Start(":80"))
}
