package config

import (
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
)

// CommandLineArgs represents the command line arguments passed to the server
type CommandLineArgs struct {
	Source   string // Type of configuration file (Command Line | Enviornment Variable | Default)
	Protocol string // Protocol to use when loading the configuration (MONGODB | FILE)
	Location string // URI of the configuration file
	Setup    bool   // If TRUE, then the server will run in SETUP mode
}

// GetCommandLineArgs returns the location of the configuration file
func GetCommandLineArgs() CommandLineArgs {

	var location string
	var setup bool

	// Look for the configuration location in the command line arguments
	pflag.StringVar(&location, "config", "", "Path to configuration file")
	pflag.BoolVar(&setup, "setup", false, "Run setup server")
	pflag.Parse()

	if location != "" {

		log.Debug().Msg("Locating configuration from command line argument.")

		return CommandLineArgs{
			Source:   ConfigSourceCommandLine,
			Location: location,
			Protocol: getConfigProtocol(location),
			Setup:    setup,
		}
	}

	// Look for the configuration location in the environment
	if location := os.Getenv("EMISSARY_CONFIG"); location != "" {

		log.Debug().Msg("Locating configuration from environment variable.")

		return CommandLineArgs{
			Source:   ConfigSourceEnvironment,
			Location: location,
			Protocol: getConfigProtocol(location),
			Setup:    setup,
		}
	}

	// Use default location
	log.Debug().Msg("No configuration specified. Using default location: `file://./config.json`")

	return CommandLineArgs{
		Source:   ConfigSourceDefault,
		Location: "file://./config.json",
		Protocol: getConfigProtocol("file://"),
		Setup:    setup,
	}
}

// getConfigProtocol returns the protocol used to load the configuration
func getConfigProtocol(location string) string {

	switch {

	case strings.HasPrefix(location, "mongodb://"):
		return StorageTypeMongo

	case strings.HasPrefix(location, "mongodb+srv://"):
		return StorageTypeMongo

	case strings.HasPrefix(location, "file://"):
		return StorageTypeFile
	}

	// Fatal error
	log.Error().Msg("Invalid configuration location.  Must be file:// or mongodb:// or mongodb+srv://")
	os.Exit(1)

	return ""
}
