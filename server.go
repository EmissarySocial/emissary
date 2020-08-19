package main

import (
	"context"
	"fmt"

	"github.com/benpate/data/mongodb"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/routes"
	"github.com/benpate/ghost/service"
	"github.com/benpate/ghost/service/templateSource"
	"github.com/davecgh/go-spew/spew"
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

	// Datasource saves db connection information.
	datasource, err := mongodb.New(viper.GetString("dbserver"), viper.GetString("dbname"))

	if err != nil {
		panic(err)
	}

	// FactoryMaker knows how to make new factories for each user request.
	factoryMaker := service.NewFactoryMaker(datasource)

	e := routes.New(factoryMaker)

	// TODO: this must be moved to DB Startup before launch
	factory := factoryMaker.Factory(context.TODO())
	templateService := factory.Template()
	directories := viper.Get("templates")

	switch value := directories.(type) {
	case string:
		fileSource := templateSource.NewFile(value)
		if err := templateService.AddSource(fileSource); err != nil {
			derp.Report(err)
		}
		spew.Dump(templateService.Templates)

	case []string:
		for _, value := range value {
			fileSource := templateSource.NewFile(value)
			templateService.AddSource(fileSource)
		}
	}

	fmt.Println("Starting web server..")

	e.Logger.Fatal(e.Start(":80"))
}
