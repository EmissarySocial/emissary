package main

import (
	"fmt"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/config"
	"github.com/benpate/ghost/route"
	"github.com/benpate/ghost/server"
)

func main() {

	fmt.Println("Starting GHOST")
	fmt.Println("Loading configuration file...")

	c, err := config.Load("./config.json")

	if err != nil {
		derp.Report(err)
		return
	}

	fmt.Println("Initializing hosts...")

	factoryManager := server.NewFactoryManager(c)

	if factoryManager.DomainCount() == 0 {
		fmt.Println("No Domains Configured!!")
		return
	}

	fmt.Println("Initializing web server...")
	e := route.New(factoryManager)

	fmt.Println("Starting web server..")
	e.Logger.Fatal(e.Start(":80"))
}
