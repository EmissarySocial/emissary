package handler

import (
	"github.com/EmissarySocial/emissary/domain"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
)

/******************************************
 * ActivityPubRouter
 * This is a modified version of the Hannibal Router.
 * I've renamed the objects to fit into this package
 * and added the domain.Factory to the RouteHandler
 * function signature, so that objects can interact
 * with the database.
 ******************************************/

// ActivityPubRouter is a simple object that routes incoming ActivityPub activities to the appropriate handler
type ActivityPubRouter struct {
	routes map[string]ActivityPubRouteHandler
}

// ActivityPubRouteHandler is a function that handles a specific type of ActivityPub activity.
// ActivityPubRouteHandlers are registered with the Router object along with the names of the activity
// types that they correspond to.
type ActivityPubRouteHandler func(factory *domain.Factory, activity streams.Document) error

// NewActivityPubRouter creates a new Router object
func NewActivityPubRouter() ActivityPubRouter {
	return ActivityPubRouter{
		routes: make(map[string]ActivityPubRouteHandler),
	}
}

// Add puts a new route to the router.  You can use "*" as a wildcard for
// either the activityType or objectType. The Handler method tries to match
// handlers from most specific to least specific.
// activity/object
// activity/*
// */object
// */*
//
// For performance reasons, this function is not thread-safe.
// So, you should add all routes before starting the server, for
// instance, in your app's `init` functions.
func (router *ActivityPubRouter) Add(activityType string, objectType string, routeHandler ActivityPubRouteHandler) {
	router.routes[activityType+"/"+objectType] = routeHandler
}

// Handle takes an ActivityPub activity and routes it to the appropriate handler
func (router *ActivityPubRouter) Handle(factory *domain.Factory, activity streams.Document) error {

	activityType := activity.Type()
	objectType := activity.Object().Type()

	if routeHandler, ok := router.routes[activityType+"/"+objectType]; ok {
		return routeHandler(factory, activity)
	}

	if routeHandler, ok := router.routes[activityType+"/*"]; ok {
		return routeHandler(factory, activity)
	}

	if routeHandler, ok := router.routes["*/"+objectType]; ok {
		return routeHandler(factory, activity)
	}

	if routeHandler, ok := router.routes["*/*"]; ok {
		return routeHandler(factory, activity)
	}

	return derp.NewBadRequestError("pub.Router.Handle", "No route found for activity", activity.Value())
}
