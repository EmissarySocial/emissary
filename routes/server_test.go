package routes

import (
	"testing"

	"github.com/benpate/ghost/service"
)

func TestServer(t *testing.T) {
	startTestServer()
}

func startTestServer() {

	// Create Test Datastore
	ds := getTestDatastore()

	// Bind Datastore Services
	factoryMaker := service.NewFactoryMaker(ds)

	e := New(factoryMaker)

	e.Start(":8080")
}
