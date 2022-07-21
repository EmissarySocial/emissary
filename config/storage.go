package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/benpate/derp"
	"github.com/spf13/pflag"
)

type Storage interface {
	Subscribe() <-chan Config
	Write(Config) error
}

// GetStorage initializes the storage engine for the server configuration
func Load(args CommandLineArgs) Storage {

	fmt.Println("Loading configuration from: " + args.Location)

	switch args.Protocol {

	case "MONGODB":
		return NewMongoStorage(args)

	case "FILE":
		return NewFileStorage(args)
	}

	// This should never happen because we've already checked this error when we parsed the command line
	derp.Report(derp.NewInternalError("config.Load", "Unable to determine storage engine", args))
	panic("Invalid configuration location.  Must be file:// or mongodb:// or mongodb+srv://")
}

// getConfigLocation returns the location of the configuration file
func getConfigLocation() (string, string) {

	// Look for the configuration location in the command line arguments
	location := pflag.String("config", "", "Path to configuration file")
	pflag.Parse()

	if (location != nil) && (*location != "") {
		return "Command Line", *location
	}

	// Look for the configuration location in the environment
	if location := os.Getenv("EMISSARY_CONFIG"); location != "" {
		return "Environment Variable", location
	}

	// Use default location
	return "Default", "file://./config.json"
}

// Write saves the current configuration to permanent storage (currently filesystem)
func Write(config Config, filename string) error {

	output, err := json.MarshalIndent(config, "", "\t")

	if err != nil {
		return derp.Wrap(err, "config.Write", "Error marshalling configuration")
	}

	if err := os.WriteFile(filename, output, 0x777); err != nil {
		return derp.Wrap(err, "config.Write", "Error writing configuration")
	}

	return nil
}
