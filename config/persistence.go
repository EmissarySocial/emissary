package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/benpate/derp"
	"github.com/spf13/pflag"
)

// GetStorage initializes the storage engine for the server configuration
func Load() Storage {

	bootstrap, location := getConfigLocation()

	fmt.Println("Loading configuration from: " + location)

	switch {

	case strings.HasPrefix(location, "mongodb://"):
		return NewMongoStorage(bootstrap, location)

	case strings.HasPrefix(location, "mongodb+srv://"):
		return NewMongoStorage(bootstrap, location)

	case strings.HasPrefix(location, "file://"):
		return NewFileStorage(bootstrap, strings.TrimPrefix(location, "file://"))
	}

	derp.Report(derp.NewInternalError("config.Load", "Unable to determine storage engine for: "+location))
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
