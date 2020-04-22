package routes

import (
	"github.com/benpate/ghost/handler"
	"github.com/benpate/ghost/service"
	"github.com/benpate/presto"
	"github.com/benpate/presto/scope"
	"github.com/labstack/echo/v4"
)

// New returns all of the routes required for this application
func New(factoryMaker service.FactoryMaker) *echo.Echo {

	e := echo.New()

	// Well-Known API calls
	// https://en.wikipedia.org/wiki/List_of_/.well-known/_services_offered_by_webservers

	e.GET("/.well-known/webfinger", handler.GetWebfinger(factoryMaker))
	e.GET("/.well-known/nodeinfo", handler.GetNodeInfo(factoryMaker))

	// Home Page for the website (should probably be a redirect to a "default" space?)
	e.GET("/", handler.TBD)

	// Home Pages for users and spaces
	e.GET("/:username", handler.TBD)
	e.GET("/:username/:pagename", handler.TBD)

	// ActivityPub
	e.GET("/inbox/:username", handler.TBD)
	e.POST("/inbox/:username", handler.TBD)
	e.GET("/outbox/:username", handler.TBD)
	e.POST("/outbox/:username", handler.TBD)

	// Presto Global Settings
	presto.UseRouter(e)
	presto.UseScopes(scope.NotDeleted)

	presto.NewCollection(factoryMaker.Stream, "/streams").
		UseScopes().
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factoryMaker.Post, "/streams/:stream/posts").
		UseScopes(scope.String("stream")).
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factoryMaker.Attachment, "/streams/:stream/pages/:page/attachments").
		UseScopes(scope.String("stream", "page")).
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factoryMaker.Comment, "/streams/:stream/pages/:page/comments").
		UseScopes(scope.String("stream", "page")).
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factoryMaker.User, "/users/:user").
		UseScopes().
		List().
		Post().
		Get().
		Put().
		Delete()

	return e
}
