package routes

import "github.com/benpate/ghost/service"

func startTestServer() {

	// Create Test Datastore
	ds := getTestDatastore()

	// Bind Datastore Services
	factoryMaker := service.NewFactoryMaker(ds)

	e := New(factoryMaker)

	e.Start(":8080")
}
