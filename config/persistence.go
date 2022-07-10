package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/benpate/derp"
)

// GetStorage initializes the storage engine for the server configuration
func Load() Storage {

	location := getConfigLocation()

	fmt.Println("Loading configuration from: " + location)

	switch {

	case strings.HasPrefix(location, "mongodb://"):
		return NewMongoStorage(location)

	case strings.HasPrefix(location, "mongodb+srv://"):
		return NewMongoStorage(location)

	case strings.HasPrefix(location, "file://"):
		return NewFileStorage(strings.TrimPrefix(location, "file://"))
	}

	derp.Report(derp.NewInternalError("config.Load", "Unable to determine storage engine for: "+location))
	panic("Invalid configuration location.  Must be file:// or mongodb:// or mongodb+srv://")
}

// getConfigLocation returns the location of the configuration file
func getConfigLocation() string {

	// Look for the configuration location in the command line arguments
	location := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	if (location != nil) && (*location != "") {
		return *location
	}

	// Look for the configuration location in the environment
	if location := os.Getenv("EMISSARY_CONFIG"); location != "" {
		return location
	}

	// Use default location
	return "file://./config.json"
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
