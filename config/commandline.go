package config

import (
	"os"
	"strings"

	"github.com/benpate/derp"
	"github.com/spf13/pflag"
)

// CommandLineArgs represents the command line arguments passed to the server
type CommandLineArgs struct {
	Source     string // Type of configuration file (Command Line | Enviornment Variable | Default)
	Protocol   string // Protocol to use when loading the configuration (MONGODB | FILE)
	Location   string // URI of the configuration file
	Initialize bool   // If TRUE, the try to initialize the configuration database
	Setup      bool   // If TRUE, then the server will run in setup mode
}

// GetCommandLineArgs returns the location of the configuration file
func GetCommandLineArgs() CommandLineArgs {

	// Look for the configuration location in the command line arguments
	location := pflag.String("config", "", "Path to configuration file")
	setup := pflag.Bool("setup", false, "Run setup server")
	initialize := pflag.Bool("init", false, "Initialize the database")

	pflag.Parse()

	if (location != nil) && (*location != "") {
		return CommandLineArgs{
			Source:     "Command Line",
			Location:   *location,
			Protocol:   getConfigProtocol(*location),
			Initialize: *initialize,
			Setup:      *setup,
		}
	}

	// Look for the configuration location in the environment
	if location := os.Getenv("EMISSARY_CONFIG"); location != "" {
		return CommandLineArgs{
			Source:     "Environment Variable",
			Location:   location,
			Protocol:   getConfigProtocol(location),
			Initialize: *initialize,
			Setup:      *setup,
		}
	}

	// Use default location
	return CommandLineArgs{
		Source:     "Default",
		Location:   "file://./config.json",
		Protocol:   getConfigProtocol("file://"),
		Initialize: *initialize,
		Setup:      *setup,
	}
}

// getConfigProtocol returns the protocol used to load the configuration
func getConfigProtocol(location string) string {

	switch {
	case strings.HasPrefix(location, "mongodb://"):
		return "MONGODB"

	case strings.HasPrefix(location, "mongodb+srv://"):
		return "MONGODB"

	case strings.HasPrefix(location, "file://"):
		return "FILE"
	}

	derp.Report(derp.NewInternalError("config.getConfigProtocol", "Unable to determine storage engine for: ", location))
	os.Exit(1)
	panic("Invalid configuration location.  Must be file:// or mongodb:// or mongodb+srv://")
}
