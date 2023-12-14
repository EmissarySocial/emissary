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
	HTTPPort int    // Port to use in setup mode (only)
}

// GetCommandLineArgs returns the location of the configuration file
func GetCommandLineArgs() CommandLineArgs {

	var source string
	var location string
	var setup bool
	var httpPort int

	// Look for the configuration location in the command line arguments
	pflag.StringVar(&location, "config", "", "Path to configuration file")
	pflag.BoolVar(&setup, "setup", false, "Run setup server")
	pflag.IntVar(&httpPort, "port", 0, "HTTP Port to use for setup mode.")
	pflag.Parse()

	if location != "" {

		// Use command line argument for configuration
		log.Debug().Msg("Locating configuration from command line argument.")
		source = ConfigSourceCommandLine

	} else if env := os.Getenv("EMISSARY_CONFIG"); env != "" {

		// Look for the configuration location in the environment
		log.Debug().Msg("Locating configuration from environment variable.")
		source = ConfigSourceEnvironment
		location = env

	} else {

		// Fall through to using default location (file in local directory)
		log.Debug().Msg("No configuration specified. Using default location: `file://./config.json`")
		location = "file://./config.json"
		source = ConfigSourceDefault
	}

	return CommandLineArgs{
		Source:   source,
		Location: location,
		Protocol: getConfigProtocol(location),
		Setup:    setup,
		HTTPPort: httpPort,
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

// ConfigOptions returns any config modifiers specified in the command line (like --port)
func (args CommandLineArgs) ConfigOptions() []Option {

	result := make([]Option, 0)

	if args.HTTPPort != 0 {
		result = append(result, WithHTTPPort(args.HTTPPort))
	}

	return result
}
