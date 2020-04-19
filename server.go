package main

import (
	"fmt"
	"net/http"

	"github.com/benpate/data/mongodb"
	"github.com/benpate/ghost/scope"
	"github.com/benpate/ghost/service"
	"github.com/benpate/presto"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func main() {

	fmt.Println("Starting GHOST")
	fmt.Println("Connecting to database...")

	// Read configuration file
	viper.SetConfigFile("./config.json")

	if err := viper.ReadInConfig(); err != nil {
		panic("Error reading configuration file: " + err.Error())
	}

	datasource := mongodb.New(viper.GetString("dbserver"), viper.GetString("dbname"))

	factory := service.NewFactory(datasource)

	e := echo.New()

	placeholder := func(ctx echo.Context) error {
		ctx.String(http.StatusOK, spew.Sdump(ctx.Request()))
		return nil
	}

	// Home Page for the website (should probably be a redirect to a "default" space?)
	e.GET("/", placeholder)

	// Home Pages for users and spaces
	e.GET("/:username", placeholder)
	e.GET("/:username/:pagename", placeholder)

	// ActivityPub
	e.GET("/inbox/:username", placeholder)
	e.POST("/inbox/:username", placeholder)
	e.GET("/outbox/:username", placeholder)
	e.POST("/outbox/:username", placeholder)

	// Presto Global Settings
	presto.UseRouter(e.)

	presto.NewCollection(factory.Presto("Stream"), "/streams").
		UseScope(scope.NotDeleted).
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factory.Presto("Post"), "/streams/:stream/posts").
		UseScopes(scope.String("stream"), scope.NotDeleted).
		List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factory.Presto("Attachment"), "/streams/:stream/pages/:page/attachments").
		UseScopes(scope.String("stream", "page"), scope.NotDeleted)
	List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factory.Presto("Comment"), "/streams/:stream/pages/:page/comments").
		UseScopes(scope.String("stream", "page"), scope.NotDeleted)
	List().
		Post().
		Get().
		Put().
		Delete()

	presto.NewCollection(factory.Presto("User"), "/users/:user").
		UseScopes(scope.NotDeleted)
	List().
		Post().
		Get().
		Put().
		Delete()

	fmt.Println("Starting web server..")

	e.Logger.Fatal(e.Start(":80"))
}
