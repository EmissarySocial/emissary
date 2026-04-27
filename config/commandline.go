package config

import (
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
)

// CommandLineArgs represents the command line arguments passed to the server
type CommandLineArgs struct {
	Source     string // Type of configuration file (Command Line | Enviornment Variable | Default)
	Location   string // URI of the configuration file
	Database   string // Name of the MongoDB config database
	Collection string // Name of the MongoDB config collection
	Setup      bool   // If TRUE, then the server will run in SETUP mode
	HTTPPort   int    // Port to use in setup mode (only)
}

// GetCommandLineArgs returns the location of the configuration file
func GetCommandLineArgs() CommandLineArgs {

	var source string
	var location string
	var db string
	var collection string
	var setup bool
	var httpPort int

	// Look for the configuration location in the command line arguments
	pflag.StringVar(&location, "config", "", "Path to configuration file")
	pflag.StringVar(&db, "db", "emissary", "Name of the MongoDB config database")
	pflag.StringVar(&collection, "collection", "config", "Name of the MongoDB config collection")
	pflag.BoolVar(&setup, "setup", false, "Run setup server")
	pflag.IntVar(&httpPort, "port", 0, "HTTP Port to use for setup mode.")
	pflag.Parse()

	if location != "" {

		// Use command line argument for configuration
		log.Info().Msg("Locating configuration from command line argument.")
		source = ConfigSourceCommandLine

	} else if env := os.Getenv("EMISSARY_CONFIG"); env != "" {

		// Look for the configuration location in the environment
		log.Info().Msg("Locating configuration from environment variable.")
		source = ConfigSourceEnvironment
		location = env

		if envDb := os.Getenv("EMISSARY_CONFIG_DB"); envDb != "" {
			db = envDb

		}

		if envCollection := os.Getenv("EMISSARY_CONFIG_COLLECTION"); envCollection != "" {
			collection = envCollection
		}

	} else {

		// Fall through to using default location (file in local directory)
		log.Info().Msg("No configuration specified. Using default location: `file://./config.json`")
		location = "file://./config.json"
		source = ConfigSourceDefault
	}

	return CommandLineArgs{
		Source:     source,
		Location:   location,
		Database:   db,
		Collection: collection,
		Setup:      setup,
		HTTPPort:   httpPort,
	}
}

// Protocol returns the protocol used to load the configuration
func (args CommandLineArgs) Protocol() string {

	switch {

	case strings.HasPrefix(args.Location, "mongodb://"):
		return StorageTypeMongo

	case strings.HasPrefix(args.Location, "mongodb+srv://"):
		return StorageTypeMongo

	case strings.HasPrefix(args.Location, "file://"):
		return StorageTypeFile
	}

	// Fatal error
	log.Error().Msg("Invalid configuration location.  Must be file:// or mongodb:// or mongodb+srv://")
	os.Exit(1)

	return ""
}

func (args CommandLineArgs) ConfigDatabase() string {

	if strings.HasPrefix(args.Location, "file://") {
		return ""
	}

	if location, err := url.Parse(args.Location); err == nil {
		location.Path = strings.TrimPrefix(location.Path, "/")
		if location.Path != "" {
			return location.Path
		}
	}

	return "emissary"
}

// ConfigOptions returns any config modifiers specified in the command line (like --port)
func (args CommandLineArgs) ConfigOptions() []Option {

	result := make([]Option, 0)

	if args.HTTPPort != 0 {
		result = append(result, WithHTTPPort(args.HTTPPort))
	}

	return result
}
