package main

import (
	"fmt"

	"github.com/benpate/data/mongodb"
	"github.com/benpate/ghost/routes"
	"github.com/benpate/ghost/service"
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
	datasource := mongodb.New(viper.GetString("dbserver"), viper.GetString("dbname"))

	// FactoryMaker knows how to make new factories for each user request.
	factoryMaker := service.NewFactoryMaker(datasource)

	e := routes.New(factoryMaker)

	fmt.Println("Starting web server..")

	e.Logger.Fatal(e.Start(":80"))
}
